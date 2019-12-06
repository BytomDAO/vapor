package txbuilder

import (
	"github.com/bytom/vapor/consensus"
	"github.com/bytom/vapor/consensus/segwit"
	"github.com/bytom/vapor/protocol/bc/types"
)

// EstimateTxGasInfo estimate transaction consumed gas
type EstimateTxGasInfo struct {
	TotalNeu    int64 `json:"total_neu"`
	FlexibleNeu int64 `json:"flexible_neu"`
	StorageNeu  int64 `json:"storage_neu"`
	VMNeu       int64 `json:"vm_neu"`
}

// EstimateTxGas estimate consumed neu for transaction
func EstimateTxGas(template Template) (*EstimateTxGasInfo, error) {
	var baseP2WSHSize, totalWitnessSize, baseP2WSHGas, totalP2WPKHGas, totalP2WSHGas int64
	baseSize := int64(176) // inputSize(112) + outputSize(64)
	baseP2WPKHSize := int64(98)
	baseP2WPKHGas := int64(1409)
	for pos, input := range template.Transaction.TxData.Inputs {
		switch input.InputType() {
		case types.SpendInputType:
			controlProgram := input.ControlProgram()
			if segwit.IsP2WPKHScript(controlProgram) {
				totalWitnessSize += baseP2WPKHSize
				totalP2WPKHGas += baseP2WPKHGas
			} else if segwit.IsP2WSHScript(controlProgram) {
				baseP2WSHSize, baseP2WSHGas = estimateP2WSHGas(template.SigningInstructions[pos])
				totalWitnessSize += baseP2WSHSize
				totalP2WSHGas += baseP2WSHGas
			}
		}
	}

	flexibleGas := int64(0)
	if totalP2WPKHGas > 0 {
		flexibleGas += baseP2WPKHGas + (baseSize+baseP2WPKHSize)*consensus.ActiveNetParams.StorageGasRate
	} else if totalP2WSHGas > 0 {
		flexibleGas += baseP2WSHGas + (baseSize+baseP2WSHSize)*consensus.ActiveNetParams.StorageGasRate
	}

	// the total transaction storage gas
	totalTxSizeGas := (int64(template.Transaction.TxData.SerializedSize) + totalWitnessSize) * consensus.ActiveNetParams.StorageGasRate

	// the total transaction gas is composed of storage and virtual machines
	totalGas := totalTxSizeGas + totalP2WPKHGas + totalP2WSHGas + flexibleGas
	if totalGas > consensus.ActiveNetParams.DefaultGasCredit {
		totalGas -= consensus.ActiveNetParams.DefaultGasCredit
	} else {
		totalGas = 0
	}

	return &EstimateTxGasInfo{
		TotalNeu:    totalGas * consensus.ActiveNetParams.VMGasRate,
		FlexibleNeu: flexibleGas * consensus.ActiveNetParams.VMGasRate,
		StorageNeu:  totalTxSizeGas * consensus.ActiveNetParams.VMGasRate,
		VMNeu:       (totalP2WPKHGas + totalP2WSHGas) * consensus.ActiveNetParams.VMGasRate,
	}, nil
}

// estimateP2WSH return the witness size and the gas consumed to execute the virtual machine for P2WSH program
func estimateP2WSHGas(sigInst *SigningInstruction) (int64, int64) {
	var witnessSize, gas int64
	for _, witness := range sigInst.WitnessComponents {
		switch t := witness.(type) {
		case *SignatureWitness:
			witnessSize += 33*int64(len(t.Keys)) + 65*int64(t.Quorum)
			gas += 1131*int64(len(t.Keys)) + 72*int64(t.Quorum) + 659
			if int64(len(t.Keys)) == 1 && int64(t.Quorum) == 1 {
				gas += 27
			}
		case *RawTxSigWitness:
			witnessSize += 33*int64(len(t.Keys)) + 65*int64(t.Quorum)
			gas += 1131*int64(len(t.Keys)) + 72*int64(t.Quorum) + 659
			if int64(len(t.Keys)) == 1 && int64(t.Quorum) == 1 {
				gas += 27
			}
		}
	}
	return witnessSize, gas
}
