// Copyright 2021 PingCAP, Inc.
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

package kv

import (
	"context"

	"google.golang.org/grpc"
)

type sharedConn struct {
	*grpc.ClientConn
	active int64
}

// GrpcPool defines an interface that can serve as a gPRC connection pool.
// It provides API to get a shared connection from pool and API to decrease usage
// reference of the shared connection
type GrpcPool interface {
	// GetConn returns an available gRPC ClientConn
	GetConn(ctx context.Context, target string, tableID int64) (*sharedConn, error)

	// ReleaseConn is called when a gRPC stream is released
	ReleaseConn(sc *sharedConn, target string, tableID int64)

	// Close tears down all ClientConns maintained in pool
	Close()
}
