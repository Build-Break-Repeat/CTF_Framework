#To run: python3 injectflags.py #
#(Replace # with number of teams)
#docker exec #######-challenge cat /flags/team#.txt
#(Replace # with Container name and team number)

#!/usr/bin/env python3
import docker
import os
import sys

#Connect to Docker
client = docker.from_env()

#Map challenge names to container names
containers = {
    'web_dvwa': 'dvwa-challenge',
    'sqli_labs': 'sqli-labs',
    'juice_shop': 'juice-shop',
    'webgoat': 'webgoat',
    'bwapp': 'bwapp',
    'nowasp': 'nowasp',
    'metasploitable2': 'metasploitable2'
}

#Get number of teams (default to 1)
num_teams = int(sys.argv[1]) if len(sys.argv) > 1 else 1

print(f"\nInjecting flags for {num_teams} teams...\n")

#For each team
for team in range(1, num_teams + 1):
    print(f"Team {team}:")
    
    #For each challenge
    for challenge_name, container_name in containers.items():
        
        #Read the flag file
        flag_file = f"flags/{challenge_name}_team{team}.txt"
        
        if not os.path.exists(flag_file):
            print(f"{challenge_name} - flag file not found")
            continue
        
        with open(flag_file, 'r') as f:
            flag = f.read().strip()
        
        #Get the container
        try:
            container = client.containers.get(container_name)
        except:
            print(f"{challenge_name} - container not running")
            continue
        
        #Put flag in container
        try:
            #Try to create /flags directory and write flag
            container.exec_run(['sh', '-c', f'mkdir -p /flags && echo "{flag}" > /flags/team{team}.txt'])
            print(f"SUCCESS - {challenge_name} -> /flags/team{team}.txt")
        except:
            print(f"FAILED - {challenge_name}")
    
    print()

print("Done\n")