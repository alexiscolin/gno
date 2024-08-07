package main

import (
	"std"

	"gno.land/p/demo/grc/exts/vault"
	"gno.land/p/demo/grc/grc20"
	"gno.land/p/demo/testutils"
	"gno.land/p/demo/ufmt"
)

func main() {
	alice := testutils.TestAddress("alice")
	bob := testutils.TestAddress("bob")       // recovery request address (cold wallet).
	charly := testutils.TestAddress("charly") // recovery dest.
	pkgaddr := std.GetOrigPkgAddr()

	// create a fooAdminToken + fooToken (GRC20) pair.
	fooAdminToken := grc20.NewAdminToken("Foo", "FOO", 4)
	fooAdminToken.Mint(alice, 1000)
	fooToken := fooAdminToken.GRC20()

	printBalances := func() {
		aliceBalance, _ := fooToken.BalanceOf(alice)
		bobBalance, _ := fooToken.BalanceOf(bob)
		charlyBalance, _ := fooToken.BalanceOf(charly)
		pkgBalance, _ := fooToken.BalanceOf(pkgaddr)
		println(ufmt.Sprintf(
			"balances: alice=%d, bob=%d, charly=%d, pkg=%d, height=%d",
			aliceBalance, bobBalance, charlyBalance, pkgBalance, std.GetHeight(),
		))
	}

	// create a vault for fooAdminToken.
	v := vault.New(fooAdminToken)
	printBalances()

	// alice deposits 300 with an unlock duration of 5 blocks.
	std.TestSetOrigCaller(alice)
	lockAmount := uint(300)
	lockDuration := uint(5)
	checkErr(v.Deposit(lockAmount, bob, lockDuration))
	printBalances()

	// alice calls unvault for 200 tokens.
	checkErr(v.Unvault(200))
	printBalances()

	// alice waits for few blocks.
	std.TestSkipHeights(int64(lockDuration) + 1)
	printBalances()

	// alice redeems 200 tokens.
	checkErr(v.Redeem())
	printBalances()

	// bob instantly recover everything in the wallet.
	std.TestSetOrigCaller(bob)
	checkErr(v.Recover(charly))
	printBalances()
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

// Output:
// balances: alice=1000, bob=0, charly=0, pkg=0, height=123
// balances: alice=700, bob=0, charly=0, pkg=300, height=123
// AAA1 123 0 5
// AAA2 123 128 5
// balances: alice=700, bob=0, charly=0, pkg=300, height=123
// balances: alice=700, bob=0, charly=0, pkg=300, height=129
// AAA3 129 128 5
// balances: alice=900, bob=0, charly=0, pkg=100, height=129
// balances: alice=900, bob=0, charly=0, pkg=100, height=129
