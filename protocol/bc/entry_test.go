package bc

import (
	"reflect"
	"testing"
)

func BenchmarkEntryID(b *testing.B) {
	m := NewMux([]*ValueSource{{Position: 1}}, &Program{Code: []byte{1}, VmVersion: 1})

	entries := []Entry{
		m,
		NewTxHeader(1, 1, 0, nil),
		NewIntraChainOutput(&ValueSource{}, &Program{Code: []byte{1}, VmVersion: 1}, 0),
		NewRetirement(&ValueSource{}, 1),
		NewSpend(&Hash{}, 0),
	}

	for _, e := range entries {
		name := reflect.TypeOf(e).Elem().Name()
		b.Run(name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				EntryID(e)
			}
		})
	}
}

func TestEntryID(t *testing.T) {
	cases := []struct {
		entry         Entry
		expectEntryID string
	}{
		{
			entry: NewMux(
				[]*ValueSource{
					{
						Ref:      &Hash{V0: 0, V1: 1, V2: 2, V3: 3},
						Value:    &AssetAmount{&AssetID{V0: 1, V1: 2, V2: 3, V3: 4}, 100},
						Position: 1,
					},
				},
				&Program{VmVersion: 1, Code: []byte{1, 2, 3, 4}},
			),
			expectEntryID: "16c4265a8a90916434c2a904a90132c198c7ebf8512aa1ba4485455b0beff388",
		},
		{
			entry: NewIntraChainOutput(
				&ValueSource{
					Ref:      &Hash{V0: 4, V1: 5, V2: 6, V3: 7},
					Value:    &AssetAmount{&AssetID{V0: 1, V1: 1, V2: 1, V3: 1}, 10},
					Position: 10,
				},
				&Program{VmVersion: 1, Code: []byte{5, 5, 5, 5}},
				1,
			),
			expectEntryID: "c60faad6ae44b15d54a57b5bd021f6cec0e5f7d2c55f53b90d6231ce5c561e9c",
		},
		{
			entry: NewRetirement(
				&ValueSource{
					Ref:      &Hash{V0: 4, V1: 5, V2: 6, V3: 7},
					Value:    &AssetAmount{&AssetID{V0: 1, V1: 1, V2: 1, V3: 1}, 10},
					Position: 10,
				},
				1,
			),
			expectEntryID: "538c367f7b6e1e9bf205ed0a29def84a1467c477b19812a6934e831c78c4da62",
		},
		{
			entry: NewCrossChainInput(
				&Hash{V0: 0, V1: 1, V2: 2, V3: 3},
				&Program{VmVersion: 1, Code: []byte{5, 5, 5, 5}},
				1,
				&AssetDefinition{
					IssuanceProgram: &Program{VmVersion: 1, Code: []byte{1, 2, 3, 4}},
					Data:            &Hash{V0: 0, V1: 1, V2: 2, V3: 3},
				},
				[]byte{},
			),
			expectEntryID: "14bb3f6e68f37d037b1f1539a21ab41e182b8d59d703a1af6c426d52cfc775d9",
		},
		{
			entry: NewCrossChainOutput(
				&ValueSource{
					Ref:      &Hash{V0: 4, V1: 5, V2: 6, V3: 7},
					Value:    &AssetAmount{&AssetID{V0: 1, V1: 1, V2: 1, V3: 1}, 10},
					Position: 10,
				},
				&Program{VmVersion: 1, Code: []byte{5, 5, 5, 5}},
				1,
			),
			expectEntryID: "8e212555174bb8b725d7023cbe1864408c6a586389875ea0143257c2402b3be9",
		},
		{
			entry: NewVoteOutput(
				&ValueSource{
					Ref:      &Hash{V0: 4, V1: 5, V2: 6, V3: 7},
					Value:    &AssetAmount{&AssetID{V0: 1, V1: 1, V2: 1, V3: 1}, 10},
					Position: 10,
				},
				&Program{VmVersion: 1, Code: []byte{5, 5, 5, 5}},
				1,
				[]byte("vote"),
			),
			expectEntryID: "67e722b339e58604e46b5a08b9684ab8b6dcb3e6218954db133c05eb2b76f0e8",
		},
		{
			entry:         NewVetoInput(&Hash{V0: 0, V1: 1, V2: 2, V3: 3}, 1),
			expectEntryID: "a4f4909f947977b50bdd978fcd320161b66a266833546b6399f4709b8dd6ad59",
		},
		{
			entry:         NewSpend(&Hash{V0: 0, V1: 1, V2: 2, V3: 3}, 1),
			expectEntryID: "2761dbb13967af8944620c134e0f336bbbb26f61eb4ecd154bc034ad6155b9e8",
		},
		{
			entry:         NewTxHeader(1, 100, 1000, []*Hash{&Hash{V0: 4, V1: 5, V2: 6, V3: 7}}),
			expectEntryID: "ba592aa0841bd4649d9a04309e2e8497ac6f295a847cadd9de6b6f9c2d806663",
		},
	}

	for _, c := range cases {
		entryID := EntryID(c.entry)
		if entryID.String() != c.expectEntryID {
			t.Errorf("the got extry id:%s is not equals to expect entry id:%s", entryID.String(), c.expectEntryID)
		}
	}
}
