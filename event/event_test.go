// Copyright (c) 2019. Baidu Inc. All Rights Reserved.

// package event is related to event subscribe.
package event

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateBlockFilter(t *testing.T) {
	filter, err := NewBlockFilter("xuper", WithEventName("event_name"),
		WithBlockRange("1", "100"),
		WithExcludeTx(true),
		WithContract("test.wasm"),
		WithAuthRequire("auth_require"),
		WithExcludeTxEvent(true),
		WithInitiator("initiator"),
		WithFromAddr("from_addr"),
		WithToAddr("to_addr"),
	)
	if err != nil {
		t.Fatalf("create block filter err: %v\n", err)
	}

	require.Equal(t, "event_name", filter.GetEventName())
	require.Equal(t, "1", filter.GetRange().GetStart())
	require.Equal(t, "100", filter.GetRange().GetEnd())
	require.Equal(t, true, filter.GetExcludeTx())
	require.Equal(t, "test.wasm", filter.GetContract())
	require.Equal(t, "auth_require", filter.GetAuthRequire())
	require.Equal(t, true, filter.GetExcludeTxEvent())
	require.Equal(t, "initiator", filter.GetInitiator())
	require.Equal(t, "from_addr", filter.GetFromAddr())
	require.Equal(t, "to_addr", filter.GetToAddr())
}
