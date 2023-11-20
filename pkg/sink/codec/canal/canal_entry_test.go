// Copyright 2022 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package canal

import (
	"testing"

	"github.com/golang/protobuf/proto"
<<<<<<< HEAD
	mm "github.com/pingcap/tidb/pkg/parser/model"
	"github.com/pingcap/tidb/pkg/parser/mysql"
=======
	mm "github.com/pingcap/tidb/parser/model"
	"github.com/pingcap/tidb/parser/mysql"
	"github.com/pingcap/tidb/types"
	"github.com/pingcap/tidb/util/rowcodec"
>>>>>>> 4a3762cdc5 (codec(ticdc): canal-json support compatible content by output detailed mysql type information (#10014))
	"github.com/pingcap/tiflow/cdc/model"
	"github.com/pingcap/tiflow/pkg/config"
	"github.com/pingcap/tiflow/pkg/sink/codec/common"
	"github.com/pingcap/tiflow/pkg/sink/codec/internal"
	"github.com/pingcap/tiflow/pkg/sink/codec/utils"
	canal "github.com/pingcap/tiflow/proto/canal"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding/charmap"
)

<<<<<<< HEAD
func TestGetMySQLTypeAndJavaSQLType(t *testing.T) {
	t.Parallel()
	canalEntryBuilder := newCanalEntryBuilder()
	for _, item := range testColumnsTable {
		obtainedMySQLType := getMySQLType(item.column)
		require.Equal(t, item.expectedMySQLType, obtainedMySQLType)

		obtainedJavaSQLType, err := getJavaSQLType(item.column, obtainedMySQLType)
		require.Nil(t, err)
		require.Equal(t, item.expectedJavaSQLType, obtainedJavaSQLType)

		if !item.column.Flag.IsBinary() {
			obtainedFinalValue, err := canalEntryBuilder.formatValue(item.column.Value, obtainedJavaSQLType)
			require.Nil(t, err)
			require.Equal(t, item.expectedEncodedValue, obtainedFinalValue)
		}
	}
}

func TestConvertEntry(t *testing.T) {
	t.Parallel()
	testInsert(t)
	testUpdate(t)
	testDelete(t)
	testDdl(t)
}

func testInsert(t *testing.T) {
	testCaseInsert := &model.RowChangedEvent{
=======
func TestInsert(t *testing.T) {
	event := &model.RowChangedEvent{
>>>>>>> 4a3762cdc5 (codec(ticdc): canal-json support compatible content by output detailed mysql type information (#10014))
		CommitTs: 417318403368288260,
		Table: &model.TableName{
			Schema: "cdc",
			Table:  "person",
		},
		Columns: []*model.Column{
			{Name: "id", Type: mysql.TypeLong, Flag: model.PrimaryKeyFlag, Value: 1},
			{Name: "name", Type: mysql.TypeVarchar, Value: "Bob"},
			{Name: "tiny", Type: mysql.TypeTiny, Value: 255},
			{Name: "comment", Type: mysql.TypeBlob, Value: []byte("测试")},
			{Name: "blob", Type: mysql.TypeBlob, Value: []byte("测试blob"), Flag: model.BinaryFlag},
		},
		ColInfos: []rowcodec.ColInfo{
			{ID: 1, IsPKHandle: true, Ft: utils.SetFlag(types.NewFieldType(mysql.TypeLong), uint(model.PrimaryKeyFlag))},
			{ID: 2, Ft: types.NewFieldType(mysql.TypeVarchar)},
			{ID: 3, Ft: types.NewFieldType(mysql.TypeTiny)},
			{ID: 4, Ft: utils.NewTextFieldType(mysql.TypeBlob)},
			{ID: 5, Ft: utils.SetBinChsClnFlag(types.NewFieldType(mysql.TypeBlob))},
		},
	}

	codecConfig := common.NewConfig(config.ProtocolCanalJSON)
	builder := newCanalEntryBuilder(codecConfig)
	entry, err := builder.fromRowEvent(event, false)
	require.NoError(t, err)
	require.Equal(t, canal.EntryType_ROWDATA, entry.GetEntryType())
	header := entry.GetHeader()
	require.Equal(t, int64(1591943372224), header.GetExecuteTime())
	require.Equal(t, canal.Type_MYSQL, header.GetSourceType())
	require.Equal(t, event.Table.Schema, header.GetSchemaName())
	require.Equal(t, event.Table.Table, header.GetTableName())
	require.Equal(t, canal.EventType_INSERT, header.GetEventType())
	store := entry.GetStoreValue()
	require.NotNil(t, store)
	rc := &canal.RowChange{}
	err = proto.Unmarshal(store, rc)
	require.NoError(t, err)
	require.False(t, rc.GetIsDdl())
	rowDatas := rc.GetRowDatas()
	require.Equal(t, 1, len(rowDatas))

	columns := rowDatas[0].AfterColumns
	require.Equal(t, len(event.Columns), len(columns))
	for _, col := range columns {
		require.True(t, col.GetUpdated())
		switch col.GetName() {
		case "id":
			require.Equal(t, int32(internal.JavaSQLTypeINTEGER), col.GetSqlType())
			require.True(t, col.GetIsKey())
			require.False(t, col.GetIsNull())
			require.Equal(t, "1", col.GetValue())
			require.Equal(t, "int", col.GetMysqlType())
		case "name":
			require.Equal(t, int32(internal.JavaSQLTypeVARCHAR), col.GetSqlType())
			require.False(t, col.GetIsKey())
			require.False(t, col.GetIsNull())
			require.Equal(t, "Bob", col.GetValue())
			require.Equal(t, "varchar", col.GetMysqlType())
		case "tiny":
			require.Equal(t, int32(internal.JavaSQLTypeTINYINT), col.GetSqlType())
			require.False(t, col.GetIsKey())
			require.False(t, col.GetIsNull())
			require.Equal(t, "255", col.GetValue())
		case "comment":
			require.Equal(t, int32(internal.JavaSQLTypeCLOB), col.GetSqlType())
			require.False(t, col.GetIsKey())
			require.False(t, col.GetIsNull())
			require.NoError(t, err)
			require.Equal(t, "测试", col.GetValue())
			require.Equal(t, "text", col.GetMysqlType())
		case "blob":
			require.Equal(t, int32(internal.JavaSQLTypeBLOB), col.GetSqlType())
			require.False(t, col.GetIsKey())
			require.False(t, col.GetIsNull())
			s, err := charmap.ISO8859_1.NewEncoder().String(col.GetValue())
			require.NoError(t, err)
			require.Equal(t, "测试blob", s)
			require.Equal(t, "blob", col.GetMysqlType())
		}
	}
}

func TestUpdate(t *testing.T) {
	event := &model.RowChangedEvent{
		CommitTs: 417318403368288260,
		Table: &model.TableName{
			Schema: "cdc",
			Table:  "person",
		},
		Columns: []*model.Column{
			{Name: "id", Type: mysql.TypeLong, Flag: model.PrimaryKeyFlag, Value: 1},
			{Name: "name", Type: mysql.TypeVarchar, Value: "Bob"},
		},
		PreColumns: []*model.Column{
			{Name: "id", Type: mysql.TypeLong, Flag: model.PrimaryKeyFlag, Value: 2},
			{Name: "name", Type: mysql.TypeVarchar, Value: "Nancy"},
		},
		ColInfos: []rowcodec.ColInfo{
			{ID: 1, IsPKHandle: true, Ft: utils.SetFlag(types.NewFieldType(mysql.TypeLong), uint(model.PrimaryKeyFlag))},
			{ID: 2, Ft: types.NewFieldType(mysql.TypeVarchar)},
		},
	}
	codecConfig := common.NewConfig(config.ProtocolCanalJSON)
	builder := newCanalEntryBuilder(codecConfig)
	entry, err := builder.fromRowEvent(event, false)
	require.NoError(t, err)
	require.Equal(t, canal.EntryType_ROWDATA, entry.GetEntryType())

	header := entry.GetHeader()
	require.Equal(t, int64(1591943372224), header.GetExecuteTime())
	require.Equal(t, canal.Type_MYSQL, header.GetSourceType())
	require.Equal(t, event.Table.Schema, header.GetSchemaName())
	require.Equal(t, event.Table.Table, header.GetTableName())
	require.Equal(t, canal.EventType_UPDATE, header.GetEventType())
	store := entry.GetStoreValue()
	require.NotNil(t, store)
	rc := &canal.RowChange{}
	err = proto.Unmarshal(store, rc)
	require.NoError(t, err)
	require.False(t, rc.GetIsDdl())
	rowDatas := rc.GetRowDatas()
	require.Equal(t, 1, len(rowDatas))

	beforeColumns := rowDatas[0].BeforeColumns
	require.Equal(t, len(event.PreColumns), len(beforeColumns))
	for _, col := range beforeColumns {
		require.True(t, col.GetUpdated())
		switch col.GetName() {
		case "id":
			require.Equal(t, int32(internal.JavaSQLTypeINTEGER), col.GetSqlType())
			require.True(t, col.GetIsKey())
			require.False(t, col.GetIsNull())
			require.Equal(t, "2", col.GetValue())
			require.Equal(t, "int", col.GetMysqlType())
		case "name":
			require.Equal(t, int32(internal.JavaSQLTypeVARCHAR), col.GetSqlType())
			require.False(t, col.GetIsKey())
			require.False(t, col.GetIsNull())
			require.Equal(t, "Nancy", col.GetValue())
			require.Equal(t, "varchar", col.GetMysqlType())
		}
	}

	afterColumns := rowDatas[0].AfterColumns
	require.Equal(t, len(event.Columns), len(afterColumns))
	for _, col := range afterColumns {
		require.True(t, col.GetUpdated())
		switch col.GetName() {
		case "id":
			require.Equal(t, int32(internal.JavaSQLTypeINTEGER), col.GetSqlType())
			require.True(t, col.GetIsKey())
			require.False(t, col.GetIsNull())
			require.Equal(t, "1", col.GetValue())
			require.Equal(t, "int", col.GetMysqlType())
		case "name":
			require.Equal(t, int32(internal.JavaSQLTypeVARCHAR), col.GetSqlType())
			require.False(t, col.GetIsKey())
			require.False(t, col.GetIsNull())
			require.Equal(t, "Bob", col.GetValue())
			require.Equal(t, "varchar", col.GetMysqlType())
		}
	}
}

func TestDelete(t *testing.T) {
	event := &model.RowChangedEvent{
		CommitTs: 417318403368288260,
		Table: &model.TableName{
			Schema: "cdc",
			Table:  "person",
		},
		PreColumns: []*model.Column{
			{Name: "id", Type: mysql.TypeLong, Flag: model.PrimaryKeyFlag, Value: 1},
		},
		ColInfos: []rowcodec.ColInfo{
			{ID: 1, IsPKHandle: true, Ft: utils.SetFlag(types.NewFieldType(mysql.TypeLong), uint(model.PrimaryKeyFlag))},
		},
	}
	codecConfig := common.NewConfig(config.ProtocolCanalJSON)
	builder := newCanalEntryBuilder(codecConfig)
	entry, err := builder.fromRowEvent(event, false)
	require.NoError(t, err)
	require.Equal(t, canal.EntryType_ROWDATA, entry.GetEntryType())
	header := entry.GetHeader()
	require.Equal(t, event.Table.Schema, header.GetSchemaName())
	require.Equal(t, event.Table.Table, header.GetTableName())
	require.Equal(t, canal.EventType_DELETE, header.GetEventType())
	store := entry.GetStoreValue()
	require.NotNil(t, store)
	rc := &canal.RowChange{}
	err = proto.Unmarshal(store, rc)
	require.NoError(t, err)
	require.False(t, rc.GetIsDdl())
	rowDatas := rc.GetRowDatas()
	require.Equal(t, 1, len(rowDatas))

	columns := rowDatas[0].BeforeColumns
	require.Equal(t, len(event.PreColumns), len(columns))
	for _, col := range columns {
		require.False(t, col.GetUpdated())
		switch col.GetName() {
		case "id":
			require.Equal(t, int32(internal.JavaSQLTypeINTEGER), col.GetSqlType())
			require.True(t, col.GetIsKey())
			require.False(t, col.GetIsNull())
			require.Equal(t, "1", col.GetValue())
			require.Equal(t, "int", col.GetMysqlType())
		}
	}
}

func TestDDL(t *testing.T) {
	event := &model.DDLEvent{
		CommitTs: 417318403368288260,
		TableInfo: &model.TableInfo{
			TableName: model.TableName{
				Schema: "cdc", Table: "person",
			},
		},
		Query: "create table person(id int, name varchar(32), tiny tinyint unsigned, comment text, primary key(id))",
		Type:  mm.ActionCreateTable,
	}
	builder := newCanalEntryBuilder(nil)
	entry, err := builder.fromDDLEvent(event)
	require.NoError(t, err)
	require.Equal(t, canal.EntryType_ROWDATA, entry.GetEntryType())
	header := entry.GetHeader()
	require.Equal(t, event.TableInfo.TableName.Schema, header.GetSchemaName())
	require.Equal(t, event.TableInfo.TableName.Table, header.GetTableName())
	require.Equal(t, canal.EventType_CREATE, header.GetEventType())
	store := entry.GetStoreValue()
	require.NotNil(t, store)
	rc := &canal.RowChange{}
	err = proto.Unmarshal(store, rc)
	require.NoError(t, err)
	require.True(t, rc.GetIsDdl())
	require.Equal(t, event.TableInfo.TableName.Schema, rc.GetDdlSchemaName())
}
