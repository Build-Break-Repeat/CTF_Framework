#To run: python3 createflags.py #
#(Replace # with number of teams)

#!/usr/bin/env python3
import os
import sys

# List of challenges
challenges = [
    "web_dvwa",
    "sqli_labs", 
    "juice_shop",
    "webgoat",
    "bwapp",
    "nowasp",
    "metasploitable2"
]

#Get number of teams (default to 1 if not specified)
num_teams = int(sys.argv[1]) if len(sys.argv) > 1 else 1

#Create flags folder
os.makedirs("flags", exist_ok=True)

print(f"\nCreating flags for {num_teams} teams...\n")

#For each team
for team in range(1, num_teams + 1):
    print(f"Team {team}:")
    
    #For each challenge
    for challenge in challenges:
        #Make a simple flag
        flag = f"flag:{{{challenge}_team{team}}}"
        
        #Save to file
        filename = f"flags/{challenge}_team{team}.txt"
        with open(filename, 'w') as f:
            f.write(flag)
        
        print(f"  {challenge} -> {flag}")
    
    print()

print(f"Created flags for {num_teams} teams\n")