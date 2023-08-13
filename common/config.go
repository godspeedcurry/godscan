package common

var version = "1.1.3"
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

var Patterns = []string{"@", "_", "#", ""}
var Passwords = []string{"!@#QWEASD", "!@#QWEASDZXC", "!QAZ2wsx", "0", "00000", "00001", "000000", "00000000", "1", "111111", "12", "123", "123123", "123321", "123456", "123!@#qwe", "123456!a", "1234567", "12345678", "123456789", "1234567890", "123456~a", "123654", "123qwe", "123qwe!@#", "1q2w#E$R", "1q2w3e", "1q2w3e4r", "1qaz!QAZ", "1qaz2wsx", "1qaz2wsx3edc", "1qaz@WSX", "1qaz@wsx#edc", "2wsx@WSX", "654123", "654321", "666666", "8888888", "88888888", "a11111", "a123123", "a12345", "a123456", "a123456.", "A123456s!", "Aa123123", "Aa1234", "Aa1234.", "Aa12345", "Aa12345.", "Aa123456", "Aa123456!", "Aa123456789", "abc123", "abc@123", "abc123456", "admin", "admin01", "admin123", "admin123!@#", "admin@123", "Admin@123", "Change_Me", "Charge123", "manager", "P@ssw0rd", "P@ssw0rd!", "pass123", "pass@123", "Passw0rd", "password", "qazwsxedc", "qwe123", "qwe123!@#", "root", "sa123456", "shell", "sysadmin", "system", "talent", "test", "test01", "test123", "toor", "admin0", "admin1", "admin2", "adminadmin"}

var DirList = []string{".git", "admin", "admin/", "nacos", "xxl-job-admin", ".DS_Store", ".svn", "actuator/env", "api/swagger-ui.html", "geoserver/index.html"}

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

// var Outputfile = getpath() + "result.txt"
var IsSave = true
var Webport = "80,81,82,83,84,85,86,87,88,89,90,91,92,98,99,443,800,801,808,880,888,889,1000,1010,1080,1081,1082,1099,1118,1888,2008,2020,2100,2375,2379,3000,3008,3128,3505,5555,6080,6648,6868,7000,7001,7002,7003,7004,7005,7007,7008,7070,7071,7074,7078,7080,7088,7200,7680,7687,7688,7777,7890,8000,8001,8002,8003,8004,8006,8008,8009,8010,8011,8012,8016,8018,8020,8028,8030,8038,8042,8044,8046,8048,8053,8060,8069,8070,8080,8081,8082,8083,8084,8085,8086,8087,8088,8089,8090,8091,8092,8093,8094,8095,8096,8097,8098,8099,8100,8101,8108,8118,8161,8172,8180,8181,8200,8222,8244,8258,8280,8288,8300,8360,8443,8448,8484,8800,8834,8838,8848,8858,8868,8879,8880,8881,8888,8899,8983,8989,9000,9001,9002,9008,9010,9043,9060,9080,9081,9082,9083,9084,9085,9086,9087,9088,9089,9090,9091,9092,9093,9094,9095,9096,9097,9098,9099,9100,9200,9443,9448,9800,9981,9986,9988,9998,9999,10000,10001,10002,10004,10008,10010,10250,12018,12443,14000,16080,18000,18001,18002,18004,18008,18080,18082,18088,18090,18098,19001,20000,20720,21000,21501,21502,28018,20880"
var MostSensitiveWebPort = "80,443,8080"

var DefaultPorts = "21,22,80,81,135,139,443,445,1433,1521,3306,5432,6379,7001,8000,8080,8089,9000,9200,11211,27017"

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
	"12#$", "WSX", "QAZ", "EDC",
	"2wsx", "1qaz", "3edc", "1q2w3e4r", "qwert",
	"#@!", "!@#$", "!@#",
	"ABC", "abc", "qwer",
	"Aa", "aA",
	"",
}
var PrefixTop = []string{
	"@",
	"!",
	"\"",
	"123",
	"_",
	"",
}

var SeperatorTop = []string{
	"@",
	"#",
	"$",
	"_",
	"!",
	".",
	"/",
	"&",
	"+",
	"",
}

var KeywordTop = []string{
	"password",
	"pass",
	"root",
	"server",
	"qwer",
	"admin",
}

type HostInfo struct {
	Url       string
	Proxy     string
	Depth     int
	Keywords  string
	Suffix    string
	Prefix    string
	Seperator string
	UrlFile   string
	IconUrl   string
	DirBrute  bool
	Show      bool
	Full      bool
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

var DEFAULT_UA string = "Mozilla/5.0 (Macintosh; Intel Mac OS X 100) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/1.0.5005.61 Safari/537.36"

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
	Proxy      string
	ListFormat bool
	Depth      int
	Keywords   string
)
