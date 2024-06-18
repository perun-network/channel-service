package utils

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/nervosnetwork/ckb-sdk-go/v2/transaction"
	"github.com/nervosnetwork/ckb-sdk-go/v2/types"
	"log"
	"sort"
)

type TransactionWithScriptGroupsWrapper struct {
	*transaction.TransactionWithScriptGroups
}

type jsonTxWithScriptGroupsWrapper struct {
	TxView              *types.Transaction    `json:"tx_view"`
	ScriptGroupsWrapper []*ScriptGroupWrapper `json:"script_groups"`
}

type ScriptGroupWrapper struct {
	*transaction.ScriptGroup
}

type jsonScriptGroupWrapper struct {
	Script        *types.Script    `json:"script"`
	GroupType     types.ScriptType `json:"group_type"`
	InputIndices  []hexutil.Uint   `json:"input_indices"`
	OutputIndices []hexutil.Uint   `json:"output_indices"`
}

func (txg *TransactionWithScriptGroupsWrapper) MarshalJSON() ([]byte, error) {
	aux := &jsonTxWithScriptGroupsWrapper{
		TxView:              txg.TxView,
		ScriptGroupsWrapper: make([]*ScriptGroupWrapper, len(txg.ScriptGroups)),
	}

	for i, sg := range txg.ScriptGroups {
		aux.ScriptGroupsWrapper[i] = &ScriptGroupWrapper{ScriptGroup: sg}
	}

	return json.Marshal(aux)
}

func (sg *ScriptGroupWrapper) MarshalJSON() ([]byte, error) {
	aux := &jsonScriptGroupWrapper{
		Script:        sg.ScriptGroup.Script,
		GroupType:     sg.GroupType,
		InputIndices:  make([]hexutil.Uint, len(sg.InputIndices)),
		OutputIndices: make([]hexutil.Uint, len(sg.OutputIndices)),
	}
	return json.Marshal(aux)
}

func (txg *TransactionWithScriptGroupsWrapper) UnmarshalJSON(data []byte) error {
	txg.TransactionWithScriptGroups = &transaction.TransactionWithScriptGroups{}
	aux := &jsonTxWithScriptGroupsWrapper{}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	txg.TxView = aux.TxView
	txg.ScriptGroups = make([]*transaction.ScriptGroup, len(aux.ScriptGroupsWrapper))
	for i, sgw := range aux.ScriptGroupsWrapper {
		txg.ScriptGroups[i] = sgw.ScriptGroup
	}

	return nil
}

func (sg *ScriptGroupWrapper) UnmarshalJSON(data []byte) error {
	sg.ScriptGroup = &transaction.ScriptGroup{}
	aux := &jsonScriptGroupWrapper{}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	toUint32Array := func(in []hexutil.Uint) []uint32 {
		out := make([]uint32, len(in))
		for i, data := range in {
			out[i] = uint32(data)
		}
		return out
	}
	// Ensure the indices are sorted
	inputIndices := toUint32Array(aux.InputIndices)
	outputIndices := toUint32Array(aux.OutputIndices)
	sort.Slice(inputIndices, func(i, j int) bool { return inputIndices[i] < inputIndices[j] })
	sort.Slice(outputIndices, func(i, j int) bool { return outputIndices[i] < outputIndices[j] })

	// Create a new instance of ScriptGroup and populate it with the converted data
	newScriptGroup := &transaction.ScriptGroup{
		Script:        aux.Script,
		GroupType:     aux.GroupType,
		InputIndices:  toUint32Array(aux.InputIndices),
		OutputIndices: toUint32Array(aux.OutputIndices),
	}

	// Assign the new ScriptGroup instance to the embedded ScriptGroup pointer
	sg.ScriptGroup = newScriptGroup

	log.Printf("Unmarshalled ScriptGroup with InputIndices: %v and OutputIndices: %v", inputIndices, outputIndices)

	return nil

}
