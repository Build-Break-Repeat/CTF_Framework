#!/usr/bin/env python3

#usage:
#chmod +x scripts/injectflags.py
#python3 scripts/injectflags.py

import docker
import os
import tempfile

#connect to Docker
client = docker.from_env()

#map challenge names to container names
CONTAINER_MAP = {
    'web_dvwa': 'dvwa-challenge',
    'sqli_labs': 'sqli-labs',
    'juice_shop': 'juice-shop',
    'webgoat': 'webgoat',
    'bwapp': 'bwapp',
    'nowasp': 'nowasp',
    'metasploitable2': 'metasploitable2'
}

#special containers that need /tmp/flags instead of /flags
SPECIAL_CONTAINERS = ['juice-shop', 'webgoat']

#containers with no shell (need file copy method)
NO_SHELL_CONTAINERS = ['juice-shop']

##############################################################################################

def inject_flag_copy(container, flag, team):
    """Inject flag by copying file into container (for containers without shell)"""
    try:
        #create temporary file with flag
        with tempfile.NamedTemporaryFile(mode='w', delete=False, suffix='.txt') as tmp:
            tmp.write(flag)
            tmp_path = tmp.name
        
        #copy file into container
        import subprocess
        dest_path = f'/tmp/team{team}.txt'
        result = subprocess.run(
            ['docker', 'cp', tmp_path, f'{container.name}:{dest_path}'],
            capture_output=True
        )
        
        #clean up temp file
        os.unlink(tmp_path)
        
        if result.returncode == 0:
            return True, '/tmp'
        return False, '/tmp'
        
    except Exception as e:
        print(f"    Copy method error: {e}")
        return False, '/tmp'

def inject_flag(container, flag, team, container_name):
    """Try multiple methods to inject flag"""
    
    #for containers with no shell, use copy method
    if container_name in NO_SHELL_CONTAINERS:
        return inject_flag_copy(container, flag, team)
    
    #use /tmp/flags for special containers
    if container_name in SPECIAL_CONTAINERS:
        flag_dir = '/tmp/flags'
    else:
        flag_dir = '/flags'
    
    #method 1: with sh
    cmd = f'mkdir -p {flag_dir} && echo "{flag}" > {flag_dir}/team{team}.txt'
    result = container.exec_run(['sh', '-c', cmd])
    if result.exit_code == 0:
        return True, flag_dir
    
    #method 2: with bash (if sh fails)
    result = container.exec_run(['bash', '-c', cmd])
    if result.exit_code == 0:
        return True, flag_dir
    
    #method 3: without shell (if bash fails)
    try:
        container.exec_run(['mkdir', '-p', flag_dir])
        result = container.exec_run(['sh', '-c', f'echo "{flag}" > {flag_dir}/team{team}.txt'])
        if result.exit_code == 0:
            return True, flag_dir
    except:
        pass
    
    return False, flag_dir

##############################################################################################

team = 1

print("##############################################################################################")
print("\ninjecting flags into containers\n")
print("##############################################################################################")

success = 0
failed = 0

for challenge_name, container_name in CONTAINER_MAP.items():
    #read the flag file
    flag_file = f'flags/{challenge_name}_team{team}.txt'
    
    #check if flag file exists
    if not os.path.exists(flag_file):
        print(f"FAILURE {challenge_name:20} - flag file not found")
        failed += 1
        continue
    
    #read flag
    with open(flag_file, 'r') as f:
        flag = f.read().strip()
    
    #inject into container
    try:
        container = client.containers.get(container_name)
        
        result, flag_dir = inject_flag(container, flag, team, container_name)
        
        if result:
            flag_path = f"{flag_dir}/team{team}.txt"
            print(f"SUCCESS {challenge_name} -> {container_name} -> {flag_path}")
            success += 1
        else:
            print(f"FAILURE {challenge_name:20} - all injection methods failed")
            failed += 1
            
    except docker.errors.NotFound:
        print(f"FAILURE {challenge_name:20} - container not running")
        failed += 1
    except Exception as e:
        print(f"FAILURE {challenge_name:20} - error: {e}")
        failed += 1

print("##############################################################################################")
print(f"\ninjection complete: {success} success, {failed} failed\n")
print("##############################################################################################")
print("")

#print verification commands
if success > 0:
    print("can verify flags with these commands:")
    print("")
    for challenge_name, container_name in CONTAINER_MAP.items():
        if container_name in NO_SHELL_CONTAINERS:
            print(f"docker exec {container_name} node -e \"console.log(require('fs').readFileSync('/tmp/team{team}.txt','utf8'))\"")
        else:
            flag_dir = '/tmp/flags' if container_name in SPECIAL_CONTAINERS else '/flags'
            print(f"docker exec {container_name} cat {flag_dir}/team{team}.txt")
    print()

##############################################################################################