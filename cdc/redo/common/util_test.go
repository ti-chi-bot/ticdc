//  Copyright 2021 PingCAP, Inc.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  See the License for the specific language governing permissions and
//  limitations under the License.

package common

import (
	"fmt"
	"testing"

	"github.com/pingcap/tiflow/cdc/model"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestParseLogFileName(t *testing.T) {
	type arg struct {
		name string
	}
	tests := []struct {
		name         string
		args         arg
		wantTs       uint64
		wantFileType string
		wantErr      string
	}{
		{
			name: "happy row .log",
			args: arg{
				name: fmt.Sprintf(RedoLogFileFormatV1, "cp",
					"test",
					DefaultRowLogFileType, 1, uuid.NewString(), LogEXT),
			},
			wantTs:       1,
			wantFileType: DefaultRowLogFileType,
		},
		{
			name: "happy row .log",
			args: arg{
				name: fmt.Sprintf(RedoLogFileFormatV2, "cp",
					"namespace", "test",
					DefaultRowLogFileType, 1, uuid.NewString(), LogEXT),
			},
			wantTs:       1,
			wantFileType: DefaultRowLogFileType,
		},
		{
			name: "happy row .tmp",
			args: arg{
				name: fmt.Sprintf(RedoLogFileFormatV1, "cp",
					"test",
					DefaultRowLogFileType, 1, uuid.NewString(), LogEXT) + TmpEXT,
			},
			wantTs:       1,
			wantFileType: DefaultRowLogFileType,
		},
		{
			name: "happy row .tmp",
			args: arg{
				name: fmt.Sprintf(RedoLogFileFormatV2, "cp",
					"namespace", "test",
					DefaultRowLogFileType, 1, uuid.NewString(), LogEXT) + TmpEXT,
			},
			wantTs:       1,
			wantFileType: DefaultRowLogFileType,
		},
		{
			name: "happy ddl .log",
			args: arg{
				name: fmt.Sprintf(RedoLogFileFormatV1, "cp",
					"test",
					DefaultDDLLogFileType, 1, uuid.NewString(), LogEXT),
			},
			wantTs:       1,
			wantFileType: DefaultDDLLogFileType,
		},
		{
			name: "happy ddl .log",
			args: arg{
				name: fmt.Sprintf(RedoLogFileFormatV2, "cp",
					"namespace", "test",
					DefaultDDLLogFileType, 1, uuid.NewString(), LogEXT),
			},
			wantTs:       1,
			wantFileType: DefaultDDLLogFileType,
		},
		{
			name: "happy ddl .sort",
			args: arg{
				name: fmt.Sprintf(RedoLogFileFormatV2, "cp",
					"default", "test",
					DefaultDDLLogFileType, 1, uuid.NewString(), LogEXT) + SortLogEXT,
			},
			wantTs:       1,
			wantFileType: DefaultDDLLogFileType,
		},
		{
			name: "happy ddl .sort",
			args: arg{
				name: fmt.Sprintf(RedoLogFileFormatV2, "cp",
					"namespace", "test",
					DefaultDDLLogFileType, 1, uuid.NewString(), LogEXT) + SortLogEXT,
			},
			wantTs:       1,
			wantFileType: DefaultDDLLogFileType,
		},
		{
			name: "happy ddl .tmp",
			args: arg{
				name: fmt.Sprintf(RedoLogFileFormatV1, "cp",
					"test",
					DefaultDDLLogFileType, 1, uuid.NewString(), LogEXT) + TmpEXT,
			},
			wantTs:       1,
			wantFileType: DefaultDDLLogFileType,
		},
		{
			name: "happy ddl .tmp",
			args: arg{
				name: fmt.Sprintf(RedoLogFileFormatV2, "cp",
					"namespace", "test",
					DefaultDDLLogFileType, 1, uuid.NewString(), LogEXT) + TmpEXT,
			},
			wantTs:       1,
			wantFileType: DefaultDDLLogFileType,
		},
		{
			name: "happy .meta",
			args: arg{
				name: "sdfsdfsf" + MetaEXT,
			},
			wantTs:       0,
			wantFileType: DefaultMetaFileType,
		},
		{
			name: "not supported fileType",
			args: arg{
				name: "sdfsdfsf.sfsf",
			},
		},
		{
			name: "err wrong format ddl .tmp",
			args: arg{
				name: fmt.Sprintf("%s_%s_%s_%s_%d%s%s", /* a wrong format */
					"cp", "default", "test",
					DefaultDDLLogFileType, 1, uuid.NewString(), LogEXT) + TmpEXT,
			},
			wantErr: ".*bad log name*.",
		},
	}
	for _, tt := range tests {
		ts, fileType, err := ParseLogFileName(tt.args.name)
		if tt.wantErr != "" {
			require.Regexp(t, tt.wantErr, err, tt.name)
		} else {
			require.Nil(t, err, tt.name)
			require.EqualValues(t, tt.wantTs, ts, tt.name)
			require.Equal(t, tt.wantFileType, fileType, tt.name)
		}
	}
}

func TestGetChangefeedFiles(t *testing.T) {
	cases := []struct {
		fileNames  []string
		changefeed model.ChangeFeedID
		want       []string
	}{
		{
			fileNames: []string{
				"captureID_test-2_uuid1.log",
				"captureID_test-3_uuid2.log",
				"captureID_test-1_uuid3.log",
			},
			changefeed: model.DefaultChangeFeedID("test-1"),
			want: []string{
				"captureID_test-1_uuid3.log",
			},
		},
		{
			fileNames: []string{
				"captureID_n1_test-2_uuid4.log",
				"captureID_n2_test-2_uuid5.log",
				"captureID_n1_test-1_uuid6.log",
			},
			changefeed: model.ChangeFeedID{"n1", "test-2"},
			want: []string{
				"captureID_n1_test-2_uuid4.log",
			},
		},
	}

	for _, c := range cases {
		got := GetChangefeedFiles(c.fileNames, c.changefeed)
		require.Equal(t, c.want, got)
	}
}
