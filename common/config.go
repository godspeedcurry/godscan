package common

var Userdict = map[string][]string{
	"ftp":        {"ftp", "admin", "www", "web", "root", "db", "wwwroot", "data", "test", "administrator", "anonymous"},
	"mysql":      {"root", "mysql"},
	"mssql":      {"sa", "sql"},
	"smb":        {"administrator", "admin", "guest", "test", "user", "manager", "webadmin", "guest"},
	"rdp":        {"administrator", "admin", "guest"},
	"postgresql": {"postgres", "admin", "test", "web"},
	"ssh":        {"root", "admin", "ubuntu", "kali", "centos"},
	"mongodb":    {"root", "admin", "mongodb", "test", "web"},
	"oracle":     {"sys", "system", "admin", "test", "web", "orcl", "oracle", "root"},
	"mem":        {"admin", "test", "root", "web", "memcached"},
	"vnc":        {"root"},
}
var NoFinger = "No finger!!"
var Patterns = []string{"@", "_", "#", ""}
var Passwords = []string{"!@#QWEASD", "!@#QWEASDZXC", "!QAZ2wsx", "0", "00000", "00001", "000000", "00000000", "1", "111111", "12", "123", "123123", "123321", "123456", "123!@#qwe", "123!@#asd", "123!@#zxc", "123456!a", "1234567", "12345678", "123456789", "1234567890", "123456~a", "123654", "123qwe", "123qwe!@#", "1q2w#E$R", "1q2w3e", "1q2w3e4r", "1qaz!QAZ", "1qaz2wsx", "1qaz2wsx3edc", "1qaz@WSX", "1qaz@wsx#edc", "2wsx@WSX", "654123", "654321", "666666", "8888888", "88888888", "a11111", "a123123", "a12345", "a123456", "a123456.", "A123456s!", "Aa123123", "Aa1234", "Aa1234.", "Aa12345", "Aa12345.", "Aa123456", "Aa123456!", "Aa123456789", "abc123", "abc@123", "abc123456", "admin", "admin01", "admin123", "admin123!@#", "admin@123", "Admin@123", "Change_Me", "Change_Me123", "Charge123", "manager", "P@ssw0rd", "P@ssw0rd!", "pass123", "pass@123", "Passw0rd", "password", "qazwsxedc", "qwe123", "qwe123!@#", "root", "sa123456", "shell", "sysadmin", "system", "talent", "test", "test01", "test123", "toor", "admin0", "admin1", "admin2", "adminadmin", "Test@123", "Abd@1234"}

var DirList = []string{
	"..;/actuator/env", "..;/api-docs", "..;/env", "..;/swagger-ui.html", "..;/v2/api-docs", ".DS_Store", ".git/config", ".git/HEAD", ".git/index", ".svn", "/", "actuator", "actuator/env", "actuator;.js", "admin", "api", "api-docs", "api-docs/", "api-docs/index.html", "api/", "api/actuator", "api/index.html", "api/swagger-resources", "api/swagger-ui.html", "api/v2/api-docs", "apidocs/", "apidocs/index.html", "core/auth/login", "docs/", "docs/index.html", "env", "geoserver/index.html", "jeecg-boot", "mappings", "nacos", "nacos/#/", "service", "services", "site.tar.gz", "swagger-resources", "swagger-ui.html", "swagger/", "swagger/index.html", "v2/api-docs", "web.tar.gz", "www.tar.gz", "xxl-job-admin", "version", "log", "metrics", "cluster", "node", "api/v1/nodes", "pods", "v2/keys",
	"..;/actuator", "..;/..;/actuator", "..;/..;/..;/actuator", "..;/..;/..;/..;/actuator", "..;/..;/..;/..;/..;/actuator", "..;/..;/..;/..;/..;/..;/..;/..;/actuator",
}

var ImportantApi = []string{"/api/v1", "/api/user", "/api/blade-user", "/api/blade-log", "/api/diag", "/api/terminal", "/api/method", "/api/triggerSnapshot", "/api/sys", "/api/system", "/api/userrolelist", "/api/hyper", "/api/dataapp", "/api/clusters", "/api/node", "/api/resourceOperations", "/api/files", "/api/external", "/api/json", "/api/latest", "/api/rest", "/api/Software", "/api/ecode", "/api/group", "/api/project", "/api/interface", "/api/plugin", "/api/v2", "/api/client", "/api/jmeter", "/api/content", "/api/experimental", "/api/portal", "/api/switch-value", "/api/Console", "/api/dp", "/api/ec", "/api/repos", "/api/session", "/api/setup", "/api/v4", "/api/image", "/api/jsonws", "/api/attachment", "/api/empower", "/api/devices", "/api/search", "/api/portalTsLogin", "/api/swagger", "/api/hrm", "/api/virtual", "/api/admin", "/api/settings", "/api/open", "/api/directive", "/api/timelion", "/api/web"}

var PORTList = map[string]int{
	"ftp":         21,
	"ssh":         22,
	"findnet":     135,
	"netbios":     139,
	"smb":         445,
	"mssql":       1433,
	"oracle":      1521,
	"mysql":       3306,
	"rdp":         3389,
	"psql":        5432,
	"redis":       6379,
	"fcgi":        9000,
	"mem":         11211,
	"mgo":         27017,
	"ms17010":     1000001,
	"cve20200796": 1000002,
	"web":         1000003,
	"webonly":     10000031,
	"all":         0,
	"portscan":    0,
	"icmp":        0,
	"main":        0,
}

var IsSave = true
var MostSensitiveWebPort = "80,443,8080"

var DefaultPorts = []string{
	"1000", "10000-10030", "10050", "1010", "10206", "10250", "10253-10255", "1043", "10443", "1080-1082", "1099", "111", "1118", "11211",
	"1194", "12018", "123", "1234", "12345", "135-139", "138", "1433", "14390", "1443", "1516", "1521", "16080", "1701", "1723", "179",
	"18000-18004", "18008", "18080-18085", "18088", "18090", "18098", "181", "1888", "1947", "2000", "20000", "2008", "2016", "2020", "2024",
	"20443", "2049", "20720", "20880", "2100", "21000", "21443", "21500-21502", "2181", "2374-2376", "2379", "23743", "23", "25000-25005",
	"253", "26000-26010", "27017-27018", "28018", "2869", "3000", "3008", "30443", "3128", "31943", "31945", "32", "33038", "3306-3308",
	"33060-33065", "3333", "3389", "34987", "37445", "38443", "39443", "389", "4500", "4789", "4848", "4899", "49336", "49593", "2", "50", "500",
	"5000", "50050-50051", "5050", "5065", "514", "5228", "5353", "5355", "5357", "5432", "54321", "54303", "5555", "5632", "56610", "5678",
	"5985", "60443", "6060", "6080", "61227", "6379", "6443", "6648", "6783", "6881", "7070-7071", "7074", "7078", "7080", "7088", "7200", "768",
	"7680", "7687-7688", "7808", "7890", "79", "8000", "8000-8019", "800-801", "801", "8020", "8028", "8030", "8038", "8042", "8044", "8046",
	"8048", "8053", "8060", "8069-8070", "808", "8080", "8080-8099", "8089", "8099", "8100-8101", "8108", "8118", "8161", "8172", "8180-8181",
	"8200", "8222", "8244", "8258", "8280", "8288", "8300", "8360", "8443", "8448", "8484", "8472", "880", "8800", "8834", "8838", "8848",
	"8858", "886-889", "8868", "8879", "8880-8881", "8888-8890", "8983", "8989", "9000-9010", "9043", "9060", "9080-9102", "9198", "9200",
	"9300", "9443", "9448", "9786", "9800", "9981", "9986", "9988", "9998-9999",
}

var SuffixTop = []string{
	"0", "1", "2", "3", "4", "5", "6", "7", "8", "9",
	"00", "000", "0000", "00000", "000000", "01", "001", "02", "03",
	"11", "111", "1111", "11111", "111111",
	"22", "222", "2222", "22222", "222222",
	"66", "666", "6666", "66666", "666666",
	"77", "777", "7777", "77777", "777777",
	"88", "888", "8888", "88888", "888888",
	"99", "999", "9999", "99999", "999999",
	"123", "456", "789",
	"321", "654", "987",
	"147", "258", "369",
	"1234", "12345", "123456", "123654", "654321",
	"123123", "1234567", "12345678", "123456789", "1234567890",
	"98", "9876", "98765", "987654", "369", "147258",
	"admin", "adminn",
	"12345+",
	"12#$", "123!@#", "WSX", "QAZ", "EDC",
	"2wsx", "1qaz", "3edc", "1q2w3e4r", "qwert",
	"#@!", "!@#$", "!@#",
	"ABC", "abc", "qwer",
	"Aa", "aA",
	"Zz", "zZ",
	"Qq", "qQ",
}
var PrefixTop = []string{
	"@",
	"!",
	"_",
	"$",
	"`",
}

var SeparatorTop = []string{
	"!",
	"@",
	"#",
	"$",
	"%",
	"^",
	"&",
	"_",
	".",
	"+",
}

var KeywordTop = []string{
	"admin",
}

type HostInfo struct {
	Url       string
	Proxy     string
	Depth     int
	Keywords  string
	Suffix    string
	Prefix    string
	Separator string
	UrlFile   string
	IconUrl   string
	DirBrute  bool
	Show      bool
	Full      bool
	Variant   bool
	// Host      string
	// Ports     string
	// Domain    string
	// Url       string
	// Path      string
	// Timeout   int64
	// Scantype  string
	// Command   string
	// SshKey    string
	// Username  string
	// Password  string
	// Usernames []string
	// Passwords []string
	// Infostr   []string
	// Hash      string
}

type PocInfo struct {
	Num        int
	Rate       int
	Timeout    int64
	Proxy      string
	PocName    string
	PocDir     string
	Target     string
	TargetFile string
	RawFile    string
	Cookie     string
	ForceSSL   bool
	ApiKey     string
	CeyeDomain string
}

var (
	// TmpOutputfile string
	// TmpSave       bool
	// IsPing        bool
	// IsWmi         bool
	// Ping          bool
	// Pocinfo       PocInfo
	// IsWebCan      bool
	// IsBrute       bool
	// RedisFile     string
	// RedisShell    string
	// Userfile      string
	// Passfile      string
	// HostFile      string
	// PortFile      string
	// PocPath       string
	// Threads       int
	Url string
	// UrlFile       string
	// Urls          []string
	// NoPorts       string
	// NoHosts       string
	// SC            string
	// PortAdd       string
	// UserAdd       string
	// PassAdd       string
	// BruteThread   int
	// LiveTop       int
	ApiPrefix  string
	LogLevel   int
	Proxy      string
	ListFormat bool
	Depth      int
	Keywords   string
)
