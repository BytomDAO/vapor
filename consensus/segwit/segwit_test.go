package segwit

import (
	"encoding/hex"
	"testing"

	"github.com/vapor/crypto/ed25519"
	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/testutil"
)

func TestConvertProgram(t *testing.T) {
	cases := []struct {
		desc    string
		program string
		script  string
		fun     func(prog []byte) ([]byte, error)
	}{
		{
			desc:    "multi sign 2-1",
			program: "0020e402787b2bf9749f8fcdcc132a44e86bacf36780ec5df2189a11020d590533ee",
			script:  "76aa20e402787b2bf9749f8fcdcc132a44e86bacf36780ec5df2189a11020d590533ee8808ffffffffffffffff7c00c0",
			fun:     ConvertP2SHProgram,
		},
		{
			desc:    "multi sign 5-3",
			program: "00200824e931fb806bd77fdcd291aad3bd0a4493443a4120062bd659e64a3e0bac66",
			script:  "76aa200824e931fb806bd77fdcd291aad3bd0a4493443a4120062bd659e64a3e0bac668808ffffffffffffffff7c00c0",
			fun:     ConvertP2SHProgram,
		},
		{
			desc:    "single sign",
			program: "001437e1aec83a4e6587ca9609e4e5aa728db7007449",
			script:  "76ab1437e1aec83a4e6587ca9609e4e5aa728db700744988ae7cac",
			fun:     ConvertP2PKHSigProgram,
		},
	}

	for i, c := range cases {
		progBytes, err := hex.DecodeString(c.program)
		if err != nil {
			t.Fatal(err)
		}

		gotScript, err := c.fun(progBytes)
		if c.script != hex.EncodeToString(gotScript) {
			t.Errorf("case #%d (%s) got script:%s, expect script:%s", i, c.desc, hex.EncodeToString(gotScript), c.script)
		}
	}
}

func TestProgramType(t *testing.T) {
	cases := []struct {
		desc    string
		program string
		fun     func(prog []byte) bool
		yes     bool
	}{
		{
			desc:    "normal P2WPKHScript",
			program: "001437e1aec83a4e6587ca9609e4e5aa728db7007449",
			fun:     IsP2WPKHScript,
			yes:     true,
		},
		{
			desc:    "ugly P2WPKHScript",
			program: "00200824e931fb806bd77fdcd291aad3bd0a4493443a4120062bd659e64a3e0bac66",
			fun:     IsP2WPKHScript,
			yes:     false,
		},
		{
			desc:    "ugly P2WPKHScript",
			program: "51",
			fun:     IsP2WPKHScript,
			yes:     false,
		},
		{
			desc:    "normal P2WSHScript",
			program: "00200824e931fb806bd77fdcd291aad3bd0a4493443a4120062bd659e64a3e0bac66",
			fun:     IsP2WSHScript,
			yes:     true,
		},
		{
			desc:    "ugly P2WSHScript",
			program: "001437e1aec83a4e6587ca9609e4e5aa728db7007449",
			fun:     IsP2WSHScript,
			yes:     false,
		},
		{
			desc:    "ugly P2WSHScript",
			program: "51",
			fun:     IsP2WSHScript,
			yes:     false,
		},
		{
			desc:    "normal IsStraightforward",
			program: "51",
			fun:     IsStraightforward,
			yes:     true,
		},
		{
			desc:    "ugly IsStraightforward",
			program: "001437e1aec83a4e6587ca9609e4e5aa728db7007449",
			fun:     IsStraightforward,
			yes:     false,
		},
		{
			desc:    "ugly IsStraightforward",
			program: "00200824e931fb806bd77fdcd291aad3bd0a4493443a4120062bd659e64a3e0bac66",
			fun:     IsStraightforward,
			yes:     false,
		},
	}

	for i, c := range cases {
		progBytes, err := hex.DecodeString(c.program)
		if err != nil {
			t.Fatal(err)
		}

		if c.fun(progBytes) != c.yes {
			t.Errorf("case #%d (%s) got %t, expect %t", i, c.desc, c.fun(progBytes), c.yes)
		}
	}
}

func TestGetXpubsAndRequiredFromProg(t *testing.T) {
	xpubStr1 := "95a1fdf4d9c30a0daf3ef6ec475058ba09b62677ce1384e33a17d028c1755ede896ec9fd8abecf0fdef9d89bba8f0d7c2576a3e78120336584884e516e128354"
	xpubStr2 := "bfc74caeb528064b056d7d1edd2913c9fb35a1bdd921087972effeb8ceb90f152b1e03199efbfc924fd7665107914309a6dcc12930256867a94b97855b392ff5"

	xpub1 := &chainkd.XPub{}
	if err := xpub1.UnmarshalText([]byte(xpubStr1)); err != nil {
		t.Fatal(err)
	}

	xpub2 := &chainkd.XPub{}
	if err := xpub2.UnmarshalText([]byte(xpubStr2)); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		desc         string
		program      string
		wantXpubs    []ed25519.PublicKey
		wantRequired int
	}{
		{
			desc:         "one xpun",
			program:      "ae20f5f412c87794b137b9433ce7d1cbb147fe5d87ca03a81d28eaba4264d7a74fc65151ad",
			wantXpubs:    []ed25519.PublicKey{xpub1.PublicKey()},
			wantRequired: 1,
		},
		{
			desc:         "two xpun",
			program:      "ae2099dbcf35be6b199e3183c7bbfe5a89d1d13978b5e1ceacbf7779507a998ba0e120ccbbbb7c72a7f8a77f227747bc8bc1f38a76ff112f395b5ff05c002e84ccd79e5152ad",
			wantXpubs:    []ed25519.PublicKey{xpub1.PublicKey(), xpub2.PublicKey()},
			wantRequired: 1,
		},
	}

	for i, c := range cases {
		progBytes, err := hex.DecodeString(c.program)
		if err != nil {
			t.Fatal(err)
		}

		gotXpus, gotRequired, err := GetXpubsAndRequiredFromProg(progBytes)
		if err != nil {
			t.Fatal(err)
		}

		if gotRequired != c.wantRequired || testutil.DeepEqual(gotXpus, c.wantXpubs) {
			t.Errorf("case #%d (%s) got xpubs: %v, Required: %d, expect xpubs: %v,  Required: %d", i, c.desc, gotXpus, gotRequired, c.wantXpubs, c.wantRequired)

		}
	}
}
