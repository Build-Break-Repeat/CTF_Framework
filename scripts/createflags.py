#!/usr/bin/env python3

#usage:
#chmod +x scripts/createflags.py
#python3 scripts/createflags.py

import os
import secrets

challenges = [
    "web_dvwa",
    "sqli_labs",
    "juice_shop",
    "webgoat",
    "bwapp",
    "nowasp",
    "metasploitable2"
]

#create flag folder
os.makedirs("flags", exist_ok=True)

##############################################################################################
team = 1

print("##############################################################################################")
print("\ncreating flags\n")
print("##############################################################################################")

for challenge in challenges:
    flag = f"flag:{{{challenge}_team{team}}}"
    
    #write flags to flag folder
    with open(f"flags/{challenge}_team{team}.txt", "w") as f:
        f.write(flag)

    print(f"{flag}")

print("##############################################################################################")
print("\nflags created and saved to flags folder\n")
print("##############################################################################################")

##############################################################################################