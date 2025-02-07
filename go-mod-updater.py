import subprocess

insideRequireBlock = False

f = open("go.mod", "r")
lines = f.readlines()

packages = []
for line in lines:
	line = line.strip()

	if line == "require (":
		insideRequireBlock = True
		continue

	if line == ")":
		insideRequireBlock = False
		continue

	if insideRequireBlock:
		ls = line.split(" ")
		packages.append(ls[0])

for pack in packages:
	print("Install/Update package: " + pack)
	process = subprocess.Popen("go get -u " + pack, shell=True, stdout=subprocess.PIPE)
	process.wait()
