package brain

type World struct {
	Player  Player      `json:"player"`
	Servers []BitServer `json:"servers"`
}

// UpdateRam adjusts RamUsed on the named server by delta (positive = consumed, negative = freed).
func (w *World) UpdateRam(hostname string, delta float64) {
	for i := range w.Servers {
		if w.Servers[i].Hostname == hostname {
			w.Servers[i].RamUsed += delta
			return
		}
	}
}

// AddProcess appends a process entry to the named server's process list.
func (w *World) AddProcess(hostname string, proc Process) {
	for i := range w.Servers {
		if w.Servers[i].Hostname == hostname {
			w.Servers[i].Processes = append(w.Servers[i].Processes, proc)
			return
		}
	}
}

type Player struct {
	Person
	Money           float64           `json:"money"`
	NumPeopleKilled uint              `json:"numPeopleKilled"`
	Entropy         uint              `json:"entropy"`
	Karma           int               `json:"karma"`
	Location        string            `json:"location"`
	TotalPlaytime   uint              `json:"totalPlaytime"`
	Factions        []string          `json:"factions"`
	Jobs            map[string]string `json:"jobs"`
}

type Person struct {
	City   string      `json:"city"`
	Exp    Skills      `json:"exp"`
	Hp     Hp          `json:"hp"`
	Mults  Multipliers `json:"mults"`
	Skills Skills      `json:"skills"`
}

type Skills struct {
	Agility      float64 `json:"agility"`
	Charisma     float64 `json:"charisma"`
	Defense      float64 `json:"defense"`
	Dexterity    float64 `json:"dexterity"`
	Hacking      float64 `json:"hacking"`
	Intelligence float64 `json:"intelligence"`
	Strength     float64 `json:"strength"`
}

type Hp struct {
	Current uint `json:"current"`
	Max     uint `json:"max"`
}

type Process struct {
	Pid      uint   `json:"pid"`
	Filename string `json:"filename"`
	Hostname string `json:"hostname"`
	Threads  uint   `json:"threads"`
	Args     []any  `json:"args"`
}

type BitServer struct {
	Processes            []Process  `json:"processes"`
	Hostname             string     `json:"hostname"`
	Ip                   string     `json:"ip"`
	OrganizationName     string     `json:"organizationName"`
	CpuCores             uint       `json:"cpuCores"`
	MaxRam               float64    `json:"maxRam"`
	RamUsed              float64    `json:"ramUsed"`
	HasAdminRights       bool       `json:"hasAdminRights"`
	PurchasedByPlayer    bool       `json:"purchasedByPlayer"`
	IsConnectedTo        bool       `json:"isConnectedTo"`
	FtpPortOpen          bool       `json:"ftpPortOpen"`
	HttpPortOpen         bool       `json:"httpPortOpen"`
	SmtpPortOpen         bool       `json:"smtpPortOpen"`
	SqlPortOpen          bool       `json:"sqlPortOpen"`
	SshPortOpen          bool       `json:"sshPortOpen"`
	BackdoorInstalled    bool       `json:"backdoorInstalled"`
	BaseDifficulty       float64    `json:"baseDifficulty"`
	HackDifficulty       float64    `json:"hackDifficulty"`
	MinDifficulty        float64    `json:"minDifficulty"`
	MoneyAvailable       float64    `json:"moneyAvailable"`
	MoneyMax             float64    `json:"moneyMax"`
	NumOpenPortsRequired uint       `json:"numOpenPortsRequired"`
	OpenPortCount        uint       `json:"openPortCount"`
	RequiredHackingSkill uint       `json:"requiredHackingSkill"`
	ServerGrowth         float64    `json:"serverGrowth"`
}

type Multipliers struct {
	AgilityExp             float64 `json:"agilityExp"`
	Agility                float64 `json:"agility"`
	BladeburnerAnalysis    float64 `json:"bladeburnerAnalysis"`
	BladeburnerMaxStamina  float64 `json:"bladeburnerMaxStamina"`
	BladeburnerStaminaGain float64 `json:"bladeburnerStaminaGain"`
	BladeburnerSuccess     float64 `json:"bladeburnerSuccess"`
	CharismaExp            float64 `json:"charismaExp"`
	Charisma               float64 `json:"charisma"`
	CompanyRep             float64 `json:"companyRep"`
	CrimeMoney             float64 `json:"crimeMoney"`
	CrimeSuccess           float64 `json:"crimeSuccess"`
	DefenseExp             float64 `json:"defenseExp"`
	Defense                float64 `json:"defense"`
	DexterityExp           float64 `json:"dexterityExp"`
	Dexterity              float64 `json:"dexterity"`
	DnetMoney              float64 `json:"dnetMoney"`
	FactionRep             float64 `json:"factionRep"`
	HackingChance          float64 `json:"hackingChance"`
	HackingExp             float64 `json:"hackingExp"`
	HackingGrow            float64 `json:"hackingGrow"`
	HackingMoney           float64 `json:"hackingMoney"`
	HackingSpeed           float64 `json:"hackingSpeed"`
	Hacking                float64 `json:"hacking"`
	HacknetNodeCoreCost    float64 `json:"hacknetNodeCoreCost"`
	HacknetNodeLevelCost   float64 `json:"hacknetNodeLevelCost"`
	HacknetNodeMoney       float64 `json:"hacknetNodeMoney"`
	HacknetNodePurchase    float64 `json:"hacknetNodePurchase"`
	HacknetNodeRamCost     float64 `json:"hacknetNodeRamCost"`
	StrengthExp            float64 `json:"strengthExp"`
	Strength               float64 `json:"strength"`
	WorkMoney              float64 `json:"workMoney"`
}
