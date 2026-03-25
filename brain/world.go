package brain

type World struct {
	player    Player
	bitServer []BitServer
	process   []Process
}

type Player struct {
	Person
	money           int
	numPeopleKilled uint
	entropy         uint
	karma           int
	location        string
	totalPlaytime   uint
	factions        []string
	jobs            map[string]string
}

type Person struct {
	city   string
	exp    Skills
	hp     Hp
	mults  Multipliers
	skills Skills
}

type Skills struct {
	agility      uint
	charisma     uint
	defense      uint
	dexterity    uint
	hacking      uint
	intelligence uint
	strength     uint
}

type Hp struct {
	current uint
	max     uint
}

type ProcessStatus uint

const (
	ProcessSpin ProcessStatus = iota
	ProcessRunning
	ProcessDone
	ProcessFailed
)

type Process struct {
	pid      uint
	filename string
	hostname string
	threads  uint
	args     []string
	status   ProcessStatus
}

type BitServer struct {
	processes            []*Process
	hostname             string
	ip                   string
	organizationName     string
	cpuCores             uint
	maxRam               float64
	ramUsed              float64
	hasAdminRights       bool
	purchasedByPlayer    bool
	isConnectedTo        bool
	ftpPortOpen          bool
	httpPortOpen         bool
	smtpPortOpen         bool
	sqlPortOpen          bool
	sshPortOpen          bool
	backdoorInstalled    bool
	baseDifficulty       float64
	hackDifficulty       float64
	minDifficulty        float64
	moneyAvailable       float64
	moneyMax             float64
	numOpenPortsRequired uint
	openPortCount        uint
	requiredHackingSkill uint
	serverGrowth         float64
}

type Multipliers struct {
	agilityExp             float64
	agility                float64
	bladeburnerAnalysis    float64
	bladeburnerMaxStamina  float64
	bladeburnerStaminaGain float64
	bladeburnerSuccess     float64
	charismaExp            float64
	charisma               float64
	companyRep             float64
	crimeMoney             float64
	crimeSuccess           float64
	defenseExp             float64
	defense                float64
	dexterityExp           float64
	dexterity              float64
	dnetMoney              float64
	factionRep             float64
	hackingChance          float64
	hackingExp             float64
	hackingGrow            float64
	hackingMoney           float64
	hackingSpeed           float64
	hacking                float64
	hacknetNodeCoreCost    float64
	hacknetNodeLevelCost   float64
	hacknetNodeMoney       float64
	hacknetNodePurchase    float64
	hacknetNodeRamCost     float64
	strengthExp            float64
	strength               float64
	workMoney              float64
}
