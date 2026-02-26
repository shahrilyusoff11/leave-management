import urllib.request
import json

data = json.dumps({"email": "superadmin@example.com", "password": "P@ssw0rd"}).encode("utf-8")
req = urllib.request.Request("http://localhost:8080/api/v1/login", data=data, headers={"Content-Type": "application/json"})
rsp = urllib.request.urlopen(req)
res = json.loads(rsp.read().decode("utf-8"))
token = res["token"]

req2 = urllib.request.Request("http://localhost:8080/api/v1/profile", headers={"Authorization": "Bearer " + token})
rsp2 = urllib.request.urlopen(req2)
print("Profile response:", rsp2.read().decode("utf-8"))
