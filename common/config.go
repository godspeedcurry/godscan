package common

import (
	_ "embed"
)

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
	"..;/actuator/env", "..;/api-docs", "..;/env", "..;/swagger-ui.html", "..;/v2/api-docs", ".DS_Store", ".git/config", ".git/HEAD", ".git/index", ".svn", "/", "actuator", "actuator/env", "actuator;.js", "admin", "api", "api-docs", "api-docs/", "api-docs/index.html", "api/", "api/actuator", "api/index.html", "api/swagger-resources", "api/swagger-ui.html", "api/v2/api-docs", "apidocs/", "apidocs/index.html", "core/auth/login", "docs/", "docs/index.html", "env", "geoserver/index.html", "jeecg-boot", "mappings", "nacos", "nacos/", "nacos/#/", "service", "services", "site.tar.gz", "swagger-resources", "swagger-ui.html", "swagger/", "swagger/index.html", "v2/api-docs", "web.tar.gz", "www.tar.gz", "xxl-job-admin", "version", "log", "metrics", "cluster", "node", "api/v1/nodes", "pods", "v2/keys",
	"..;/actuator", "..;/..;/actuator", "..;/..;/..;/actuator", "..;/..;/..;/..;/actuator", "..;/..;/..;/..;/..;/actuator", "..;/..;/..;/..;/..;/..;/..;/..;/actuator",
}

var ImportantApi = []string{"/api/v1", "/api/user", "/api/blade-user", "/api/blade-log", "/api/diag", "/api/terminal", "/api/method", "/api/triggerSnapshot", "/api/sys", "/api/system", "/api/userrolelist", "/api/hyper", "/api/dataapp", "/api/clusters", "/api/node", "/api/resourceOperations", "/api/files", "/api/external", "/api/json", "/api/latest", "/api/rest", "/api/Software", "/api/ecode", "/api/group", "/api/project", "/api/interface", "/api/plugin", "/api/v2", "/api/client", "/api/jmeter", "/api/content", "/api/experimental", "/api/portal", "/api/switch-value", "/api/Console", "/api/dp", "/api/ec", "/api/repos", "/api/session", "/api/setup", "/api/v4", "/api/image", "/api/jsonws", "/api/attachment", "/api/empower", "/api/devices", "/api/search", "/api/portalTsLogin", "/api/swagger", "/api/hrm", "/api/virtual", "/api/admin", "/api/settings", "/api/open", "/api/directive", "/api/timelion", "/api/web", "/graphql"}

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

// database
// cloud
// web
// top 100
// top 500
// top 1000
// top 5000
// top 10000
// top 20000
var WebPorts = []string{
	"21", "22", "25", "80-88", "443",
	"10000-10030", "10250", "10443", "18080-18085", "20443", "21443", "30443",
	"3306-3308", "3389", "38443", "39443", "5000", "50050-50051", "5432", "54321",
	"5985", "60443", "6443", "8000-8019", "8020", "8080-8099", "8443", "8448",
	"8880-8881", "8888-8890", "9000-9010", "9043", "9060", "9080-9102", "9200",
	"9443", "9998-9999",
}

//go:embed ports_20000
var customProbes string

var AllPorts = customProbes
var DefaultPorts = []string{"80", "443", "2083", "8080", "7547", "2095", "22", "2078", "2096", "2087", "2077", "8443", "888", "2082", "5060", "2086", "8000", "8888", "161", "21", "8880", "53", "8089", "2052", "554", "30005", "8081", "2053", "52230", "2080", "4567", "8008", "1701", "2079", "3389", "58000", "500", "8088", "1723", "81", "2000", "123", "8085", "25", "37777", "23", "49152", "2091", "5985", "51005", "9000", "1024", "3306", "111", "5000", "7080", "8082", "47001", "7170", "8001", "6881", "49154", "49153", "139", "88", "50001", "445", "1194", "9090", "5001", "135", "1025", "49155", "8291", "110", "50995", "49665", "14440", "587", "14430", "9020", "9080", "3000", "50805", "2222", "143", "520", "993", "4433", "30010", "8090", "9200", "50996", "51001", "8015", "50999", "995", "50997", "49667", "8002", "50998", "51000", "465", "7000", "51003", "51002", "20002", "82", "51004", "1717", "49666", "8083", "19000", "49156", "5357", "49664", "9100", "8084", "7777", "8887", "9999", "10000", "49668", "5678", "3128", "52869", "6467", "6466", "10250", "8181", "9001", "49157", "58603", "9530", "37443", "10443", "444", "1026", "9010", "10001", "137", "8086", "6443", "49669", "2107", "8999", "60000", "20201", "2105", "2103", "4443", "85", "1080", "9443", "20000", "51007", "55555", "8020", "18080", "12121", "17000", "60002", "7001", "5432", "5555", "8009", "49158", "3001", "9527", "5006", "32400", "9091", "7848", "8899", "40000", "9876", "9305", "8010", "1433", "1900", "7443", "2525", "12345", "8444", "90", "10002", "6000", "1027", "50777", "8172", "10101", "8099", "8889", "9307", "9304", "4430", "6060", "5353", "8800", "8200", "50000", "2121", "4343", "9306", "9303", "83", "9002", "4444", "1883", "5523", "9003", "1500", "9998", "5900", "6379", "2323", "7081", "5683", "30006", "3333", "52200", "4040", "515", "6363", "8728", "1234", "7070", "43999", "6699", "631", "2223", "8087", "10010", "4000", "9009", "2601", "6001", "10022", "8100", "2049", "49502", "7005", "7548", "800", "3307", "541", "50580", "119", "20202", "8003", "49501", "8069", "8989", "5061", "179", "12350", "65004", "1000", "8282", "51200", "10011", "5005", "49159", "84", "6264", "3479", "3005", "646", "26", "10005", "8091", "12349", "42235", "9500", "2443", "873", "27017", "5986", "8445", "8006", "4911", "3002", "8096", "9012", "7003", "7004", "30003", "7010", "2379", "6666", "22222", "5222", "1028", "7002", "3443", "9004", "9092", "9800", "9101", "8159", "18018", "5431", "808", "9600", "999", "8005", "8787", "24442", "89", "10080", "5002", "9013", "9093", "3030", "8061", "602", "9021", "43080", "3003", "9099", "10020", "8990", "30000", "3006", "8580", "1443", "9005", "28080", "5007", "7778", "50011", "9191", "8881", "86", "8004", "8058", "3050", "8686", "50002", "38520", "8022", "8991", "9109", "6789", "91", "18443", "8383", "9030", "7071", "9444", "7800", "18017", "1201", "9103", "9088", "49161", "5080", "3702", "8123", "8060", "843", "18888", "7011", "8050", "990", "8070", "3031", "8180", "4848", "1029", "2404", "8016", "12380", "8043", "1302", "19080", "3010", "4434", "6005", "60001", "8866", "8011", "8765", "4500", "4190", "7676", "30001", "5672", "9988", "4431", "9089", "6008", "52931", "1688", "3008", "6080", "9007", "15672", "8014", "15000", "10003", "7050", "8883", "5500", "8092", "8222", "9102", "5090", "9081", "9085", "20001", "8554", "9801", "9105", "9094", "19999", "6002", "8012", "9008", "9900", "5050", "50050", "5400", "6380", "8101", "8098", "42443", "3080", "2200", "3004", "2090", "16001", "5443", "40005", "8530", "30004", "3299", "9098", "7100", "9212", "113", "3400", "98", "9062", "7500", "21242", "2196", "1935", "11001", "10009", "44444", "4800", "7999", "8023", "9095", "9991", "9663", "9308", "7019", "7020", "25565", "15001", "666", "548", "6036", "3100", "9553", "9082", "60443", "5569", "10243", "50100", "9119", "9143", "9040", "9014", "21300", "8315", "5600", "7700", "20080", "99", "2332", "8585", "9201", "8025", "9019", "5601", "2600", "8097", "14443", "50012", "12588", "8500", "16443", "30021", "7013", "8885", "4321", "9083"}

var TableHeader = []string{"Url", "Title", "Finger", "ContentType", "Status", "location", "Length", "Keyword", "SimHash"}
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
