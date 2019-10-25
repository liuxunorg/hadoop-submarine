#!/bin/bash
#
# Licensed to the Apache Software Foundation (ASF) under one
# or more contributor license agreements.  See the NOTICE file
# distributed with this work for additional information
# regarding copyright ownership.  The ASF licenses this file
# to you under the Apache License, Version 2.0 (the
# "License"); you may not use this file except in compliance
# with the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# description: Start and stop daemon script for.
#

USAGE="-e Usage: submarine-launcher.sh [start|stop] [SUBMARINE_SERVER|WORKBENCH_SERVER] id"

if [ -L ${BASH_SOURCE-$0} ]; then
  BIN=$(dirname $(readlink "${BASH_SOURCE-$0}"))
else
  BIN=$(dirname ${BASH_SOURCE-$0})
fi
export BIN=$(cd "${BIN}">/dev/null; pwd)

. "${BIN}/common.sh"

cd ${BIN}/>/dev/null

LAUNCHER_NAME="Submarine Launcher"
LAUNCHER_LOGFILE="${SUBMARINE_LOG_DIR}/launcher.log"
LAUNCHER_MAIN=org.apache.submarine.launcher.LauncherProcess
LAUNCHER_JAVA_OPTS_MERGE+=" -Dlauncher.log.file=${LAUNCHER_LOGFILE}"

add_jar_in_dir "${BIN}/lib/lancher"

function initialize_default_directories() {
  if [[ ! -d "${SUBMARINE_LOG_DIR}" ]]; then
    echo "Log dir doesn't exist, create ${SUBMARINE_LOG_DIR}"
    $(mkdir -p "${SUBMARINE_LOG_DIR}")
  fi
}

function found_launcher_pid() {
  process='LauncherProcess';
  RUNNING_PIDS=$(ps x | grep ${process} | grep -v grep | awk '{print $1}');

  if [[ -z "${RUNNING_PIDS}" ]]; then
    return
  fi

  if ! kill -0 ${RUNNING_PIDS} > /dev/null 2>&1; then
    echo "${LAUNCHER_NAME} running but process is dead"
  fi

  echo "${RUNNING_PIDS}"
}

function wait_for_launcher_to_die() {
  local pid
  local count

  pid=`found_launcher_pid`
  timeout=10
  count=0
  timeoutTime=$(date "+%s")
  let "timeoutTime+=$timeout"
  currentTime=$(date "+%s")
  forceKill=1

  while [[ $currentTime -lt $timeoutTime ]]; do
    $(kill ${pid} > /dev/null 2> /dev/null)
    if kill -0 ${pid} > /dev/null 2>&1; then
      sleep 3
    else
      forceKill=0
      break
    fi
    currentTime=$(date "+%s")
  done

  if [[ forceKill -ne 0 ]]; then
    $(kill -9 ${pid} > /dev/null 2> /dev/null)
  fi
}

function start() {
  local pid

  pid=`found_launcher_pid`
  if [[ ! -z "$pid" && "$pid" != 0 ]]; then
    echo "${LAUNCHER_NAME}:${pid} is already running"
    return 0;
  fi

  initialize_default_directories

  nohup $JAVA_RUNNER ${LAUNCHER_JAVA_OPTS_MERGE} ${LAUNCHER_MAIN} >> "${LAUNCHER_LOGFILE}" 2>&1 < /dev/null &
  pid=$!
  if [[ ! -z "${pid}" ]]; then
    echo "${LAUNCHER_NAME} start"
    return 1;
  fi
}

function stop() {
  local pid
  pid=`found_launcher_pid`

  if [[ -z "$pid" ]]; then
    echo "${LAUNCHER_NAME} is not running"
    return 0;
  else
    wait_for_launcher_to_die
    echo "${LAUNCHER_NAME} stop"
  fi
}

function find_launcher_process() {
  local pid
  pid=`found_launcher_pid`

  if [[ -z "$pid" ]]; then
    echo "${LAUNCHER_NAME} is not running"
    return 1
  else
    if ! kill -0 ${pid} > /dev/null 2>&1; then
      echo "${LAUNCHER_NAME} running but process is dead"
      return 1
    else
      echo "${LAUNCHER_NAME} is running"
    fi
  fi
}

case "${1}" in
  start)
    start
    ;;
  stop)
    stop
    ;;
  restart)
    echo "${LAUNCHER_NAME} is restarting" >> "${LAUNCHER_LOGFILE}"
    stop
    start
    ;;
  status)
    find_launcher_process
    ;;
  *)
    echo ${USAGE}
esac
