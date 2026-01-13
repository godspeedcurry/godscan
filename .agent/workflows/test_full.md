---
description: Perform a full regression test suite (port scan, spider, weakpass) against a local implementation
---
1. Setup local test server
// turbo
python3 -m http.server 8000 > /dev/null 2>&1 & echo $! > server.pid

2. Run Port Scan Test
// turbo
go run main.go port -i 127.0.0.1 -p 8000

3. Run Spider Test
// turbo
go run main.go spider -u http://127.0.0.1:8000 -d 1

4. Run Weakpass Test
// turbo
go run main.go weakpass -k admin --full > weakpass_test.txt && head -n 5 weakpass_test.txt && rm weakpass_test.txt

5. Cleanup
// turbo
kill $(cat server.pid) && rm server.pid
