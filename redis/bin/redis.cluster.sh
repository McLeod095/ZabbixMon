#!/bin/bash

CONFDIR="/tmp/zabbix"
CACHETIME="60"
MD5SUM=$(which md5sum 2>/dev/null)
REDISHOST=${2:-}
REDISPORT=${3:-}
REDISURL="${REDISHOST}:${REDISPORT}"
REDISHOSTS=()
flagCluster=1
DEBUG=0

debug() {
	[ "${DEBUG}" == "1" ] && printf "[%s] %s\n" "$(date)" "$*"
}

# Функция заглушка для заббикса
zbx() {
	printf "ZBX_NOTSUPPORTED"
	exit 1
}

# Загрузить сохраненный ранее конфигурационный файл
loadConfig() {
	debug "loadConfig"
	readarray -t REDISHOSTS < "${CONFFILE}"
	return 0
}

checkConfig() {
	debug "checkConfig"
	if [ -e "${CONFFILE}" ] && [ $(wc -l "${CONFFILE}" 2>/dev/null | cut -d" " -f1) -gt "1" ] && [ "$(( $(date +%s) - $(stat -c %Z ${CONFFILE}) ))" -le "${CACHETIME}" ]; then 
		loadConfig 
		return 0
	fi
	return 1
}

# Сохранить данные в файле
saveConfig() {
	debug "saveConfig"
	if isCluster; then
		>"${CONFFILE}"
		for host in "${REDISHOSTS[@]}"; do 
			printf "%s\n" $host >> "${CONFFILE}"
		done
	fi
}

# Наблюдаемый обюъект является кластером?
isCluster() {
	debug "isCluster = ${flagCluster}"
	return ${flagCluster}
}

# Проверка на кластер
checkForCluster() {
	local CMD
	local iscluster
	debug "checkForCluster $REDISHOST $REDISPORT"
	#iscluster=$(docker exec $(docker ps -qlf {name=redis*}) redis-cli -h ${REDISHOST} -p ${REDISPORT} cluster info | grep -c 'ERR')
	iscluster=$(redis-cli -h ${REDISHOST} -p ${REDISPORT} cluster info | grep -c 'ERR')
	debug "iscluster = ${iscluster}"

	if [ "${iscluster}" -eq "0" ];then
		flagCluster=0
	fi
	return ${flagCluster}
}

# Получить список всех нод кластера
getClusterNodes() {
	debug "getClusterNodes"
	local line
	if isCluster; then
		unset REDISHOSTS
		#mapfile -t clusterNodes < <(docker exec $(docker ps -qlf {name=redis*}) redis-cli -h "${REDISHOST}" -p "${REDISPORT}" cluster nodes 2>/dev/null| tr " " "|")
		mapfile -t clusterNodes < <(redis-cli -h "${REDISHOST}" -p "${REDISPORT}" cluster nodes 2>/dev/null| tr " " "|")
		for line in ${clusterNodes[@]}; do 
			line=$(printf "${line}" | cut -d"|" -f2)
			local h=${line%:*}
			local p=${line#*:}
			[ -z "${h}" ] && h=${REDISURL%:*}
			REDISHOSTS[${#REDISHOSTS[@]}]="${h}:${p}"
		done
	fi
}

# Поулчить информацию о кластере
getClusterInfo() {
	local param=${1:-}
	debug "getClusterInfo ${param}"
	[ -z "${param}" ] && return 1
	if isCluster; then
		#local CLUSTERINFO=$(docker exec $(docker ps -qlf {name=redis*}) redis-cli -h "${REDISHOST}" -p "${REDISPORT}" cluster info 2>/dev/null | grep -oP "${param}:\K.*")
		local CLUSTERINFO=$(redis-cli -h "${REDISHOST}" -p "${REDISPORT}" cluster info 2>/dev/null | grep -oP "${param}:\K.*")
		debug "getClusterInfo: ${CLUSTERINFO}"
		[ -n "${CLUSTERINFO}" ] && echo "${CLUSTERINFO}" || zbx
	fi
}

setHostPort() {
	debug "setHostPort"
	REDISHOST=${1%:*}
	REDISPORT=${1#*:}
}

# Проверить доступность инстанса
doPing() {
	local redishost="${1}"
	local redisport="${2}"
	debug "doPing $redishost $redisport"
	local ping
	#ping=$(docker exec $(docker ps -qlf {name=redis*}) redis-cli -h "${redishost}" -p "${redisport}" ping 2>/dev/null | grep -c 'PONG')
	ping=$(redis-cli -h "${redishost}" -p "${redisport}" ping 2>/dev/null | grep -c 'PONG')
	if [ "${ping}" == "1" ]; then
		return 0
	fi
	return 1
}

# Проверить доступность инстансов, запомнить первый ответивший
getPing() {
	debug "getPing"
	local host
	for host in "${REDISHOSTS[@]}"; do 
		local h=${host%:*}
		local p=${host#*:}
		if doPing "${h}" "${p}"; then
			REDISHOST="${h}"
			REDISPORT="${p}"
			return 0
		fi
	done
	return 1
}

# Количество master 
getMasterCount() {
	printf '%s\n' "${clusterNodes[@]}" | awk -F"|" 'BEGIN{c=0} $3~/master/{c++}END{print c}'
}

# Количество slave
getSlaveCount() {
	printf '%s\n' "${clusterNodes[@]}" | awk -F"|" 'BEGIN{c=0} $3~/slave/{c++}END{print c}'
}

# Количество в состоянии fail
getFailCount() {
	printf '%s\n' "${clusterNodes[@]}" | awk -F"|" 'BEGIN{c=0} $3!~/fail\?/&& $3~/fail/{c++}END{print c}'
}

# Количество в состоянии pfail
getPFailCount() {
	printf '%s\n' "${clusterNodes[@]}" | awk -F"|" 'BEGIN{c=0} $3~/fail\?/{c++}END{print c}'
}

# Количесвто master в состоянии fail
getMasterFailCount() {
	printf '%s\n' "${clusterNodes[@]}" | awk -F"|" 'BEGIN{c=0} $3~/master/ && $3~/fail/ && $3!~/fail\?/{c++}END{print c}'
}

#
getMasterPFailCount() {
	printf '%s\n' "${clusterNodes[@]}" | awk -F"|" 'BEGIN{c=0} $3~/master/ && $3~/fail\?/{c++}END{print c}'
}

getSlaveFailCount() {
	printf '%s\n' "${clusterNodes[@]}" | awk -F"|" 'BEGIN{c=0} $3~/slave/ && $3~/fail/ && $3!~/fail\?/{c++}END{print c}'
}

getSlavePFailCount() {
	printf '%s\n' "${clusterNodes[@]}" | awk -F"|" 'BEGIN{c=0} $3~/slave/ && $3~/fail\?/{c++}END{print c}'
}

getSlaveWithoutMaster() {
	printf '%s\n' "${clusterNodes[@]}" | awk -F"|" 'BEGIN{c=0}$3~/slave/ && $4~/-/{c++}END{print c}'
}

getConnectedState() {
	printf '%s\n' "${clusterNodes[@]}" | awk -F"|" 'BEGIN{c=0}$8=/connected/{c++}END{print c}'
}

getNotConnectedState() {
	printf '%s\n' "${clusterNodes[@]}" | awk -F"|" 'BEGIN{c=0}$8=/disconnected/{c++}END{print c}'
}

getMasterMinSlave() {
	printf '%s\n' "${clusterNodes[@]}" | awk -F"|" '$3~/slave/&&$3!~/fail/{slave[$4]++} $3~/master/&&$3!~/fail/{master[$1]++}END{min=0;for(s in slave){min=(min<slave[s]?slave[s]:min)};for (m in master){ if(slave[m]!=""){min=(min<slave[m]?min:slave[m])}} print min}'
}

getMasterNodes() {
	printf '%s\n' "${clusterNodes[@]}" | awk -F"|" '$3~/master/ && $3!~/fail/{sub(/:.*/,"",$2);master[$2]++}END{max=0; for(m in master){max=(max<master[m]?master[m]:max)} print max}'
}

[ -z "${REDISURL}" ] && exit 1

[ ! -d "${CONFDIR}" ] && mkdir -p "${CONFDIR}"

MD5URL=$(printf "${REDISURL}" | ${MD5SUM} | cut -d" " -f1)
CONFFILE="${CONFDIR}/${MD5URL}"
LOCKFILE="${CONFFILE}.lock"

exec 200>${LOCKFILE}

if ! flock -xw 5 200; then
	zbx
fi

printf '%s' "${REDISURL}" > ${LOCKFILE}

if ! checkConfig; then 
	REDISHOSTS[0]="${REDISURL}"
fi

if getPing; then
	ping=1
	checkForCluster
	getClusterNodes
	saveConfig
else
	printf "0"
	exit 0
fi

case "${1}" in
	"ping")
		printf "%s" "${ping}"
		;;
	"iscluster")
		isCluster && printf "1" || printf "0"
		;;
	"cluster_state"|"cluster_slots_assigned"|"cluster_slots_ok"|"cluster_slots_pfail"|"cluster_slots_fail"|"cluster_known_nodes"|"cluster_size"|"cluster_current_epoch"|"cluster_my_epoch"|"cluster_stats_messages_sent"|"cluster_stats_messages_received")
		getClusterInfo "${1}"
		;;
	"master")
		getMasterCount
		;;
	"slave")
		getSlaveCount
		;;
	"fail")
		getFailCount
		;;
	"pfail")
		getPFailCount
		;;
	"masterfail")
		getMasterFailCount
		;;
	"masterpfail")
		getMasterPFailCount
		;;
	"slavefail")
		getSlaveFailCount
		;;
	"slavepfail")
		getSlavePFailCount
		;;
	"slavewithoutmaster")
		getSlaveWithoutMaster
		;;
	"connected")
		getConnectedState
		;;
	"notconnected")
		getNotConnectedState
		;;
	"masterminslave")
		getMasterMinSlave
	 	;;
	"masternodes")
		getMasterNodes
		;;
	*)
		zbx
		;;
esac	
