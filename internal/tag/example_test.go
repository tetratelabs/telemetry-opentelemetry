// Copyright (c) Tetrate, Inc 2023.
// Copyright 2017, OpenCensus Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package tag

import (
	"context"
	"log"
)

var (
	tagMap *Map
	ctx    context.Context
	key    Key
)

func ExampleNewKey() {
	// Get a key to represent user OS.
	key, err := NewKey("example.com/keys/user-os")
	if err != nil {
		log.Fatal(err)
	}
	_ = key // use key
}

func ExampleMustNewKey() {
	key := MustNewKey("example.com/keys/user-os")
	_ = key // use key
}

func ExampleNew() {
	osKey := MustNewKey("example.com/keys/user-os")
	userIDKey := MustNewKey("example.com/keys/user-id")

	ctx, err := New(ctx,
		Insert(osKey, "macOS-10.12.5"),
		Upsert(userIDKey, "cde36753ed"),
	)
	if err != nil {
		log.Fatal(err)
	}

	_ = ctx // use context
}

func ExampleNew_replace() {
	ctx, err := New(ctx,
		Insert(key, "macOS-10.12.5"),
		Upsert(key, "macOS-10.12.7"),
	)
	if err != nil {
		log.Fatal(err)
	}

	_ = ctx // use context
}

func ExampleNewContext() {
	// Propagate the tag map in the current context.
	ctx := NewContext(context.Background(), tagMap)

	_ = ctx // use context
}

func ExampleFromContext() {
	tagMap := FromContext(ctx)

	_ = tagMap // use the tag map
}

func ExampleDo() {
	ctx, err := New(ctx,
		Insert(key, "macOS-10.12.5"),
		Upsert(key, "macOS-10.12.7"),
	)
	if err != nil {
		log.Fatal(err)
	}
	Do(ctx, func(ctx context.Context) {
		_ = ctx // use context
	})
}
