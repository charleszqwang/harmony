package shardingconfig

import (
	"fmt"
	"math/big"
	"testing"
)

func TestMainnetInstanceForEpoch(t *testing.T) {
	tests := []struct {
		epoch    *big.Int
		instance Instance
	}{
		{
			big.NewInt(0),
			mainnetV0,
		},
		{
			big.NewInt(12),
			mainnetV1,
		},
		{
			big.NewInt(19),
			mainnetV1_1,
		},
		{
			big.NewInt(25),
			mainnetV1_2,
		},
	}

	for _, test := range tests {
		in := MainnetSchedule.InstanceForEpoch(test.epoch)
		if in.NumShards() != test.instance.NumShards() || in.NumNodesPerShard() != test.instance.NumNodesPerShard() {
			t.Errorf("can't get the right instane for epoch: %v\n", test.epoch)
		}
	}
}

func TestCalcEpochNumber(t *testing.T) {
	tests := []struct {
		block uint64
		epoch *big.Int
	}{
		{
			0,
			big.NewInt(0),
		},
		{
			1,
			big.NewInt(0),
		},
		{
			327679,
			big.NewInt(0),
		},
		{
			327680,
			big.NewInt(0),
		},
		{
			344064,
			big.NewInt(1),
		},
		{
			344063,
			big.NewInt(0),
		},
		{
			344065,
			big.NewInt(1),
		},
		{
			360448,
			big.NewInt(2),
		},
	}

	for i, test := range tests {
		ep := MainnetSchedule.CalcEpochNumber(test.block)
		if ep.Cmp(test.epoch) != 0 {
			t.Errorf("CalcEpochNumber error: index %v, got %v, expect %v\n", i, ep, test.epoch)
		}
	}
}

func TestGetShardingStructure(t *testing.T) {
	shardID := 0
	numShard := 4
	res := genShardingStructure(numShard, shardID, "http://s%d.t.hmy.io:9500", "ws://s%d.t.hmy.io:9800")
	if len(res) != 4 || !res[0]["current"].(bool) || res[1]["current"].(bool) || res[2]["current"].(bool) || res[3]["current"].(bool) {
		t.Error("Error when generating sharding structure")
	}
	for i := 0; i < numShard; i++ {
		if res[i]["current"].(bool) != (i == shardID) {
			t.Error("Error when generating sharding structure")
		}
		if res[i]["shardID"].(int) != i {
			t.Error("Error when generating sharding structure")
		}
		if res[i]["http"].(string) != fmt.Sprintf("http://s%d.t.hmy.io:9500", i) {
			t.Error("Error when generating sharding structure")
		}
		if res[i]["ws"].(string) != fmt.Sprintf("ws://s%d.t.hmy.io:9800", i) {
			t.Error("Error when generating sharding structure")
		}
	}
}
