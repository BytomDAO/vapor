package vmutil

import "github.com/bytom/vapor/protocol/vm"

// P2WMCProgramV2 return the segwit pay to magnetic contract
func P2WMCProgramV2(magneticContractArgs MagneticContractArgs) ([]byte, error) {
	builder := NewBuilder()
	builder.AddInt64(1)
	builder.AddData(magneticContractArgs.RequestedAsset.Bytes())
	builder.AddInt64(magneticContractArgs.RatioNumerator)
	builder.AddInt64(magneticContractArgs.RatioDenominator)
	builder.AddData(magneticContractArgs.SellerProgram)
	builder.AddData(magneticContractArgs.SellerKey)
	return builder.Build()
}

// P2MCProgramV2 generates the script for control with the v2 version of magnetic contract
//
// MagneticV2 contract source code:
// contract MagneticV2(requestedAsset: Asset,
// 					   	ratioNumerator: Integer,
// 					   	ratioDenominator: Integer,
// 						sellerProgram: Program,
// 						standardProgram: Program,
// 						sellerKey: PublicKey) locks valueAmount of valueAsset {
//  clause partialTrade(exchangeAmount: Amount, fee: Amount) {
// 		define actualAmount: Integer = exchangeAmount * ratioDenominator / ratioNumerator
// 		verify actualAmount >= 0 && actualAmount < valueAmount
// 		define receiveAmount: Integer = exchangeAmount * (10000 - fee) / 10000
// 		verify receiveAmount >= 0
// 		lock receiveAmount of requestedAsset with sellerProgram
// 		lock valueAmount-actualAmount of valueAsset with standardProgram
// 		unlock actualAmount of valueAsset
// 	}
// 	clause fullTrade(fee: Amount) {
// 		define requestedAmount: Integer = valueAmount * ratioNumerator / ratioDenominator
// 		define tradeAmount: Integer = requestedAmount * (10000 - fee) / 10000
// 		verify tradeAmount >= 0
// 		lock tradeAmount of requestedAsset with sellerProgram
// 		unlock valueAmount of valueAsset
// 	}
// 	clause cancel(sellerSig: Signature) {
// 		verify checkTxSig(sellerKey, sellerSig)
// 		unlock valueAmount of valueAsset
// 	}
// }
//
// contract stack flow:
// 7                        [... <position> <clause selector> sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset 7]
// ROLL                     [... <clause selector> sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset <position>]
// TOALTSTACK               [... <clause selector> sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset]
// 6                        [... <clause selector> sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset 6]
// ROLL                     [... sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset <clause selector>]
// DUP                      [... sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset <clause selector> <clause selector>]
// 2                        [... sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset <clause selector> <clause selector> 2]
// NUMEQUAL                 [... sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset <clause selector> (<clause selector> == 2)]
// JUMPIF:$cancel           [... sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset <clause selector>]
// JUMPIF:$fullTrade        [... sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset]
// $partialTrade            [... sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset]
// 7                        [... exchangeAmount fee sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset 7]
// PICK                     [... exchangeAmount fee sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset exchangeAmount]
// 3                        [... exchangeAmount fee sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset exchangeAmount 3]
// ROLL                     [... exchangeAmount fee sellerKey standardProgram sellerProgram ratioNumerator requestedAsset exchangeAmount ratioDenominator]
// 3                        [... exchangeAmount fee sellerKey standardProgram sellerProgram ratioNumerator requestedAsset exchangeAmount ratioDenominator 3]
// ROLL                     [... exchangeAmount fee sellerKey standardProgram sellerProgram requestedAsset exchangeAmount ratioDenominator ratioNumerator]
// MULFRACTION              [... exchangeAmount fee sellerKey standardProgram sellerProgram requestedAsset actualAmount]
// AMOUNT                   [... exchangeAmount fee sellerKey standardProgram sellerProgram requestedAsset actualAmount valueAmount]
// OVER                     [... exchangeAmount fee sellerKey standardProgram sellerProgram requestedAsset actualAmount valueAmount actualAmount]
// 0                        [... exchangeAmount fee sellerKey standardProgram sellerProgram requestedAsset actualAmount valueAmount actualAmount 0]
// GREATERTHANOREQUAL       [... exchangeAmount fee sellerKey standardProgram sellerProgram requestedAsset actualAmount valueAmount (actualAmount >= 0)]
// 2                        [... exchangeAmount fee sellerKey standardProgram sellerProgram requestedAsset actualAmount valueAmount (actualAmount >= 0) 2]
// PICK                     [... exchangeAmount fee sellerKey standardProgram sellerProgram requestedAsset actualAmount valueAmount (actualAmount >= 0) actualAmount]
// ROT                      [... exchangeAmount fee sellerKey standardProgram sellerProgram requestedAsset actualAmount (actualAmount >= 0) actualAmount valueAmount]
// LESSTHAN                 [... exchangeAmount fee sellerKey standardProgram sellerProgram requestedAsset actualAmount (actualAmount >= 0) (actualAmount < valueAmount)]
// BOOLAND                  [... exchangeAmount fee sellerKey standardProgram sellerProgram requestedAsset actualAmount ((actualAmount >= 0) && (actualAmount < valueAmount))]
// VERIFY                   [... exchangeAmount fee sellerKey standardProgram sellerProgram requestedAsset actualAmount]
// 6                        [... exchangeAmount fee sellerKey standardProgram sellerProgram requestedAsset actualAmount 6]
// ROLL                     [... fee sellerKey standardProgram sellerProgram requestedAsset actualAmount exchangeAmount]
// 10000                    [... fee sellerKey standardProgram sellerProgram requestedAsset actualAmount exchangeAmount 10000]
// 7                        [... fee sellerKey standardProgram sellerProgram requestedAsset actualAmount exchangeAmount 10000 7]
// ROLL                     [... sellerKey standardProgram sellerProgram requestedAsset actualAmount exchangeAmount 10000 fee]
// SUB                      [... sellerKey standardProgram sellerProgram requestedAsset actualAmount exchangeAmount (10000 - fee)]
// 10000                    [... sellerKey standardProgram sellerProgram requestedAsset actualAmount exchangeAmount (10000 - fee) 10000]
// MULFRACTION              [... sellerKey standardProgram sellerProgram requestedAsset actualAmount receiveAmount]
// DUP                      [... sellerKey standardProgram sellerProgram requestedAsset actualAmount receiveAmount receiveAmount]
// 0                        [... sellerKey standardProgram sellerProgram requestedAsset actualAmount receiveAmount receiveAmount 0]
// GREATERTHANOREQUAL       [... sellerKey standardProgram sellerProgram requestedAsset actualAmount receiveAmount (receiveAmount >= 0)]
// VERIFY                   [... sellerKey standardProgram sellerProgram requestedAsset actualAmount receiveAmount]
// FROMALTSTACK             [... sellerKey standardProgram sellerProgram requestedAsset actualAmount receiveAmount <position>]
// DUP 						[... sellerKey standardProgram sellerProgram requestedAsset actualAmount receiveAmount <position> <position>]
// TOALTSTACK				[... sellerKey standardProgram sellerProgram requestedAsset actualAmount receiveAmount <position>]
// SWAP                     [... sellerKey standardProgram sellerProgram requestedAsset actualAmount <position> receiveAmount]
// 3                        [... sellerKey standardProgram sellerProgram requestedAsset actualAmount <position> receiveAmount 3]
// ROLL                     [... sellerKey standardProgram sellerProgram actualAmount <position> receiveAmount requestedAsset]
// 1                        [... sellerKey standardProgram sellerProgram actualAmount <position> receiveAmount requestedAsset 1]
// 5                        [... sellerKey standardProgram sellerProgram actualAmount <position> receiveAmount requestedAsset 1 5]
// ROLL                     [... sellerKey standardProgram actualAmount <position> receiveAmount requestedAsset 1 sellerProgram]
// CHECKOUTPUT              [... sellerKey standardProgram actualAmount checkOutput(receiveAmount, requestedAsset, sellerProgram)]
// VERIFY                   [... sellerKey standardProgram actualAmount]
// FROMALTSTACK				[... sellerKey standardProgram actualAmount <position>]
// 1                        [... sellerKey standardProgram actualAmount <position> 1]
// ADD                      [... sellerKey standardProgram actualAmount (<position> + 1)]
// AMOUNT                   [... sellerKey standardProgram actualAmount (<position> + 1) valueAmount]
// ROT                      [... sellerKey standardProgram (<position> + 1) valueAmount actualAmount]
// SUB                      [... sellerKey standardProgram (<position> + 1) (valueAmount - actualAmount)]
// ASSET                    [... sellerKey standardProgram (<position> + 1) (valueAmount - actualAmount) valueAsset]
// 1                        [... sellerKey standardProgram (<position> + 1) (valueAmount - actualAmount) valueAsset 1]
// 4                        [... sellerKey standardProgram (<position> + 1) (valueAmount - actualAmount) valueAsset 1 4]
// ROLL                     [... sellerKey (<position> + 1) (valueAmount - actualAmount) valueAsset 1 standardProgram]
// CHECKOUTPUT              [... sellerKey checkOutput((valueAmount - actualAmount), valueAsset, standardProgram)]
// JUMP:$_end               [... sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset]
// $fullTrade               [... sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset]
// AMOUNT                   [... fee sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset valueAmount]
// ROT                      [... fee sellerKey standardProgram sellerProgram ratioDenominator requestedAsset valueAmount ratioNumerator]
// 3                        [... fee sellerKey standardProgram sellerProgram ratioDenominator requestedAsset valueAmount ratioNumerator 3]
// ROLL                     [... fee sellerKey standardProgram sellerProgram requestedAsset valueAmount ratioNumerator ratioDenominator]
// MULFRACTION              [... fee sellerKey standardProgram sellerProgram requestedAsset requestedAmount]
// 10000                    [... fee sellerKey standardProgram sellerProgram requestedAsset requestedAmount 10000]
// 6                        [... fee sellerKey standardProgram sellerProgram requestedAsset requestedAmount 10000 6]
// ROLL                     [... sellerKey standardProgram sellerProgram requestedAsset requestedAmount 10000 fee]
// SUB                      [... sellerKey standardProgram sellerProgram requestedAsset requestedAmount (10000 - fee)]
// 10000                    [... sellerKey standardProgram sellerProgram requestedAsset requestedAmount (10000 - fee) 10000]
// MULFRACTION              [... sellerKey standardProgram sellerProgram requestedAsset tradeAmount]
// DUP                      [... sellerKey standardProgram sellerProgram requestedAsset tradeAmount tradeAmount]
// 0                        [... sellerKey standardProgram sellerProgram requestedAsset tradeAmount tradeAmount 0]
// GREATERTHANOREQUAL       [... sellerKey standardProgram sellerProgram requestedAsset tradeAmount (tradeAmount >= 0)]
// VERIFY                   [... sellerKey standardProgram sellerProgram requestedAsset tradeAmount]
// FROMALTSTACK             [... sellerKey standardProgram sellerProgram requestedAsset tradeAmount <position>]
// SWAP                     [... sellerKey standardProgram sellerProgram requestedAsset <position> tradeAmount]
// ROT                      [... sellerKey standardProgram sellerProgram <position> tradeAmount requestedAsset]
// 1                        [... sellerKey standardProgram sellerProgram <position> tradeAmount requestedAsset 1]
// 4                        [... sellerKey standardProgram sellerProgram <position> tradeAmount requestedAsset 1 4]
// ROLL                     [... sellerKey standardProgram <position> tradeAmount requestedAsset 1 sellerProgram]
// CHECKOUTPUT              [... sellerKey standardProgram checkOutput(tradeAmount, requestedAsset, sellerProgram)]
// JUMP:$_end               [... sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset]
// $cancel                  [... sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset <clause selector>]
// DROP                     [... sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset]
// 6                        [... sellerSig sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset 6]
// ROLL                     [... sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset sellerSig]
// 6                        [... sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset sellerSig 6]
// ROLL                     [... standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset sellerSig sellerKey]
// TXSIGHASH SWAP CHECKSIG  [... standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset checkTxSig(sellerKey, sellerSig)]
// $_end                    [... sellerKey standardProgram sellerProgram ratioDenominator ratioNumerator requestedAsset]
func P2MCProgramV2(magneticContractArgs MagneticContractArgs) ([]byte, error) {
	standardProgram, err := P2WMCProgramV2(magneticContractArgs)
	if err != nil {
		return nil, err
	}

	builder := NewBuilder()
	// contract arguments
	builder.AddData(magneticContractArgs.SellerKey)
	builder.AddData(standardProgram)
	builder.AddData(magneticContractArgs.SellerProgram)
	builder.AddInt64(magneticContractArgs.RatioDenominator)
	builder.AddInt64(magneticContractArgs.RatioNumerator)
	builder.AddData(magneticContractArgs.RequestedAsset.Bytes())

	// contract instructions
	builder.AddOp(vm.OP_7)
	builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_TOALTSTACK)
	builder.AddOp(vm.OP_6)
	builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_DUP)
	builder.AddOp(vm.OP_2)
	builder.AddOp(vm.OP_NUMEQUAL)
	builder.AddJumpIf(0)
	builder.AddJumpIf(1)

	builder.AddOp(vm.OP_7)
	builder.AddOp(vm.OP_PICK)
	builder.AddOp(vm.OP_3)
	builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_3)
	builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_MULFRACTION)
	builder.AddOp(vm.OP_AMOUNT)
	builder.AddOp(vm.OP_OVER)
	builder.AddOp(vm.OP_0)
	builder.AddOp(vm.OP_GREATERTHANOREQUAL)
	builder.AddOp(vm.OP_2)
	builder.AddOp(vm.OP_PICK)
	builder.AddOp(vm.OP_ROT)
	builder.AddOp(vm.OP_LESSTHAN)
	builder.AddOp(vm.OP_BOOLAND)
	builder.AddOp(vm.OP_VERIFY)

	builder.AddOp(vm.OP_6)
	builder.AddOp(vm.OP_ROLL)
	builder.AddInt64(10000)
	builder.AddOp(vm.OP_7)
	builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_SUB)
	builder.AddInt64(10000)
	builder.AddOp(vm.OP_MULFRACTION)
	builder.AddOp(vm.OP_DUP)
	builder.AddOp(vm.OP_0)
	builder.AddOp(vm.OP_GREATERTHANOREQUAL)
	builder.AddOp(vm.OP_VERIFY)

	builder.AddOp(vm.OP_FROMALTSTACK)
	builder.AddOp(vm.OP_DUP)
	builder.AddOp(vm.OP_TOALTSTACK)
	builder.AddOp(vm.OP_SWAP)
	builder.AddOp(vm.OP_3)
	builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_1)
	builder.AddOp(vm.OP_5)
	builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_CHECKOUTPUT)

	builder.AddOp(vm.OP_VERIFY)
	builder.AddOp(vm.OP_FROMALTSTACK)
	builder.AddOp(vm.OP_1)
	builder.AddOp(vm.OP_ADD)
	builder.AddOp(vm.OP_AMOUNT)
	builder.AddOp(vm.OP_ROT)
	builder.AddOp(vm.OP_SUB)
	builder.AddOp(vm.OP_ASSET)
	builder.AddOp(vm.OP_1)
	builder.AddOp(vm.OP_4)
	builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_CHECKOUTPUT)

	builder.AddJump(2)
	builder.SetJumpTarget(1)
	builder.AddOp(vm.OP_AMOUNT)
	builder.AddOp(vm.OP_ROT)
	builder.AddOp(vm.OP_3)
	builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_MULFRACTION)
	builder.AddInt64(10000)
	builder.AddOp(vm.OP_6)
	builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_SUB)
	builder.AddInt64(10000)
	builder.AddOp(vm.OP_MULFRACTION)
	builder.AddOp(vm.OP_DUP)
	builder.AddOp(vm.OP_0)
	builder.AddOp(vm.OP_GREATERTHANOREQUAL)
	builder.AddOp(vm.OP_VERIFY)
	builder.AddOp(vm.OP_FROMALTSTACK)
	builder.AddOp(vm.OP_SWAP)
	builder.AddOp(vm.OP_ROT)
	builder.AddOp(vm.OP_1)
	builder.AddOp(vm.OP_4)
	builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_CHECKOUTPUT)

	builder.AddJump(3)
	builder.SetJumpTarget(0)
	builder.AddOp(vm.OP_DROP)
	builder.AddOp(vm.OP_6)
	builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_6)
	builder.AddOp(vm.OP_ROLL)
	builder.AddOp(vm.OP_TXSIGHASH)
	builder.AddOp(vm.OP_SWAP)
	builder.AddOp(vm.OP_CHECKSIG)
	builder.SetJumpTarget(2)
	builder.SetJumpTarget(3)
	return builder.Build()
}
