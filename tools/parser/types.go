package parser

type EthLog map[string]interface{}

type controlAddress struct {
	Owner        string   `json:"owner"`
	Worker       string   `json:"worker"`
	ControlAddrs []string `json:"controlAddrs"`
}

type execParams struct {
	CodeCid           string `json:"CodeCid"`
	ConstructorParams string `json:"constructorParams"`
}

type beneficiaryTerm struct {
	Quota      string `json:"quota"`
	UsedQuota  string `json:"usedQuota"`
	Expiration int64  `json:"expiration"`
}
type activeBeneficiary struct {
	Beneficiary string          `json:"beneficiary"`
	Term        beneficiaryTerm `json:"term"`
}

type proposed struct {
	NewBeneficiary        string `json:"newBeneficiary"`
	NewQuota              string `json:"newQuota"`
	NewExpiration         int64  `json:"newExpiration"`
	ApprovedByBeneficiary bool   `json:"approvedByBeneficiary"`
	ApprovedByNominee     bool   `json:"approvedByNominee"`
}

type getBeneficiryReturn struct {
	Active   activeBeneficiary `json:"active"`
	Proposed proposed          `json:"proposed"`
}
