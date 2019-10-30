#!/bin/bash
#
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

export DEFAULT_MYSQL_VERSION=5.1.39

if [[ -L ${BASH_SOURCE-$0} ]]; then
  FWDIR=$(dirname $(readlink "${BASH_SOURCE-$0}"))
else
  FWDIR=$(dirname "${BASH_SOURCE-$0}")
fi

if [[ -z "${SUBMARINE_HOME}" ]]; then
  # Make SUBMARINE_HOME look cleaner in logs by getting rid of the
  # extra ../
  export SUBMARINE_HOME="$(cd "${FWDIR}/.."; pwd)"
fi

if [[ -z "${SUBMARINE_CONF_DIR}" ]]; then
  export SUBMARINE_CONF_DIR="${SUBMARINE_HOME}/conf"
fi

if [[ -z "${SUBMARINE_LOG_DIR}" ]]; then
  export SUBMARINE_LOG_DIR="${SUBMARINE_HOME}/logs"
fi

if [[ -f "${SUBMARINE_CONF_DIR}/submarine-env.sh" ]]; then
  . "${SUBMARINE_CONF_DIR}/submarine-env.sh"
fi

WORKBENCH_CLASSPATH+=":${SUBMARINE_CONF_DIR}"
LAUNCHE_CLASSPATH+=":${SUBMARINE_CONF_DIR}"

function add_each_jar_in_dir(){
  if [[ -d "${1}" ]]; then
    for jar in $(find -L "${1}" -maxdepth 1 -name '*jar'); do
      WORKBENCH_CLASSPATH="${jar}:${WORKBENCH_CLASSPATH}"
      LAUNCHE_CLASSPATH="${jar}:${LAUNCHE_CLASSPATH}"
    done
  fi
}

function add_each_jar_in_dir_recursive(){
  if [[ -d "${1}" ]]; then
    for jar in $(find -L "${1}" -type f -name '*jar'); do
      WORKBENCH_CLASSPATH="${jar}:${WORKBENCH_CLASSPATH}"
      LAUNCHE_CLASSPATH="${jar}:${LAUNCHE_CLASSPATH}"
    done
  fi
}

function add_jar_in_dir(){
  if [[ -d "${1}" ]]; then
    WORKBENCH_CLASSPATH="${1}/*:${WORKBENCH_CLASSPATH}"
    LAUNCHE_CLASSPATH="${1}/*:${LAUNCHE_CLASSPATH}"
  fi
}

function download_mysql_jdbc_jar(){
  if [[ -z "${MYSQL_JAR_URL}" ]]; then
    if [[ -z "${MYSQL_VERSION}" ]]; then
      MYSQL_VERSION="${DEFAULT_MYSQL_VERSION}"
    fi
    MYSQL_JAR_URL="https://repo1.maven.org/maven2/mysql/mysql-connector-java/${MYSQL_VERSION}/mysql-connector-java-${MYSQL_VERSION}.jar"
  fi
  echo "Downloading mysql jdbc jar from ${MYSQL_JAR_URL}."
  wget ${MYSQL_JAR_URL} -P "${BIN}/../workbench/lib"
  echo "Mysql jdbc jar is downloaded and put in the path of workbench/lib."
}

WORKBENCH_JAVA_OPTS_MERGE+=" ${WORKBENCH_JAVA_OPTS} -Dfile.encoding=UTF-8 ${WORKBENCH_MEM}"
WORKBENCH_JAVA_OPTS_MERGE+=" -Dlog4j.configuration=file://${SUBMARINE_CONF_DIR}/log4j.properties"
export WORKBENCH_JAVA_OPTS_MERGE

LAUNCHER_JAVA_OPTS_MERGE+=" ${LAUNCHER_JAVA_OPTS} -Dfile.encoding=UTF-8 ${LAUNCHER_MEM}"
LAUNCHER_JAVA_OPTS_MERGE+=" -Dlog4j.configuration=file://${SUBMARINE_CONF_DIR}/log4j.properties"
export LAUNCHER_JAVA_OPTS_MERGE

if [[ -n "${JAVA_HOME}" ]]; then
  JAVA_RUNNER="${JAVA_HOME}/bin/java"
else
  JAVA_RUNNER=java
fi
export JAVA_RUNNER
