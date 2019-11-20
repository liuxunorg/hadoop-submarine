/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package org.apache.submarine.commons.metastore;

import org.apache.hadoop.hive.metastore.api.InvalidObjectException;
import org.apache.hadoop.hive.metastore.api.MetaException;
import org.apache.submarine.commons.utils.SubmarineConfiguration;
import org.junit.Test;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.sql.Connection;
import java.sql.DriverManager;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.sql.Statement;
import static org.junit.Assert.assertTrue;

public class SubmarineMetaStoreTest {
  private static Logger LOG = LoggerFactory.getLogger(SubmarineMetaStoreTest.class);

  private static final SubmarineConfiguration submarineConf = SubmarineConfiguration.getInstance();

  @Test
  public void createDatabase() throws InvalidObjectException, MetaException {
    System.out.println("createDatabase <<<1 ");

    String url = "jdbc:mysql://127.0.0.1:3306/metastoredb_test?" +
        "useUnicode=true&amp;characterEncoding=UTF-8&amp;autoReconnect=true&amp;" +
        "failOverReadOnly=false&amp;zeroDateTimeBehavior=convertToNull&amp;useSSL=false";
    String username = "metastore_test";
    String password = "password_test";
    boolean flag = false;
    Connection con = null;
    Statement stmt = null;
    try {
      con = DriverManager.getConnection(url, username, password);
      stmt = con.createStatement();
      String sql = "show tables";
      System.out.println("sql:" + sql);
      ResultSet rs = stmt.executeQuery(sql);
      System.out.println("rs:" + rs);

      while (rs.next()) {
        String pass = rs.getString(1);
        System.out.println("table:" + pass);
      }
    } catch (SQLException se) {
      se.printStackTrace();
      LOG.info("123= {} {}", se.getErrorCode(), se.getMessage());
    }

    System.out.println("createDatabase >>>2 ");
    assertTrue(false);
  }
}
