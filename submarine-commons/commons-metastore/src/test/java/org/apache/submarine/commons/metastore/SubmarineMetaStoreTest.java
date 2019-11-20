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

import org.apache.hadoop.hive.metastore.api.Database;
import org.apache.hadoop.hive.metastore.api.FieldSchema;
import org.apache.hadoop.hive.metastore.api.InvalidObjectException;
import org.apache.hadoop.hive.metastore.api.MetaException;
import org.apache.hadoop.hive.metastore.api.NoSuchObjectException;
import org.apache.hadoop.hive.metastore.api.PrincipalType;
import org.apache.hadoop.hive.metastore.api.SerDeInfo;
import org.apache.hadoop.hive.metastore.api.StorageDescriptor;
import org.apache.hadoop.hive.metastore.api.Table;
import org.apache.submarine.commons.utils.SubmarineConfiguration;
import org.junit.After;
import org.junit.Before;
import org.junit.Test;

import java.sql.Connection;
import java.sql.DriverManager;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.sql.Statement;
import java.util.ArrayList;
import java.util.Date;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import static org.junit.Assert.assertEquals;

public class SubmarineMetaStoreTest {
  private static final SubmarineConfiguration submarineConf = SubmarineConfiguration.getInstance();

  @Before
  public void createDatabase() throws InvalidObjectException, MetaException {
    System.out.println("createDatabase <<< ");

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
      System.out.println("数据库连接失败！");
    }

    System.out.println("createDatabase >>> ");
  }
}
