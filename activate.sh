#!/bin/bash
# activation script for phonebook

HOST=localhost
PORT=8250
DEVTESTING=0

DBNAME="accord"
DBUSER="ec2-user"
IAM=$(whoami)

usage() {
    cat <<ZZEOF
Phonebook activation script.
Usage:   activate.sh [OPTIONS] CMD

OPTIONS:
-p port      (default is 8250)
-h hostname  (default is localhost)
-N dbname    (default is accord)
-T           (use this option to indicate testing rather than production)

CMD is one of: start | stop | ready | test | teststatus

cmd is case insensitive

Examples:
Command to start phonebook:
	bash$  activate.sh START 

Command to start phonebook for testing purposes:
	bash$  activate.sh -T START 

	If you do testing in the context of the source code tree and you don't 
	use -T, you may see messages like this:

	  ./activate.sh: line 101: ./pbwatchdog: No such file or directory

Command to stop phonebook:
	bash$  activate.sh Stop

Command to see if phonebook is ready for commands... the response
will be "OK" if it is ready, or something else if there are problems:

    bash$  activate.sh ready
    OK
ZZEOF
	exit 0
}

updateImages() {
	/usr/local/accord/bin/getfile.sh jenkins-snapshot/phonebook/latest/pbimages.tar.gz
	gunzip pbimages.tar.gz
	tar xvf pbimages.tar
}

stopwatchdog() {
	# make sure we can find it
    pidline=$(ps -ef | grep pbwatchdog | grep -v grep)
    if [ "${pidline}" != "" ]; then
        lines=$(echo "${pidline}" | wc -l)
        if [ $lines -gt "0" ]; then
            pid=$(echo "${pidline}" | awk '{print $2}')
            $(kill $pid)
        fi          
    fi      
}

while getopts ":p:ih:N:T" o; do
    case "${o}" in
       h)
            HOST=${OPTARG}
            echo "HOST set to: ${HOST}"
            ;;
        N)
            DBNAME=${OPTARG}
            echo "DBNAME set to: ${DBNAME}"
            ;;
        p)
            PORT=${OPTARG}
	    	echo "PORT set to: ${PORT}"
            ;;
        T)
            DEVTESTING=1
	    	echo "DEVTESTING set to: ${DEVTESTING}"
            ;;
        *)
            usage
            ;;
    esac
done
shift $((OPTIND-1))

for arg do
	# echo '--> '"\`$arg'"
	cmd=$(echo ${arg}|tr "[:upper:]" "[:lower:]")
    case "$cmd" in
    "images")
		updateImages
		echo "Images updated"
		;;
	"start")
		#===============================================
		# START
		# Add the command to start your application...
		#===============================================
		if [ ${IAM} == "root" ]; then
			chown -R ec2-user *
			chmod u+s phonebook pbwatchdog
		fi

		if [ "${DEVTESTING}" -ne "1" ]; then
			if [ ! -d "./images" ]; then
				/usr/local/accord/bin/getfile.sh jenkins-snapshot/phonebook/latest/pbimages.tar.gz >phonebook.log 2>&1
				gunzip -f pbimages.tar.gz >phonebook.log 2>&1
				tar xvf pbimages.tar >phonebook.log 2>&1
			fi
			if [ ! -f "/usr/local/share/man/man1/pbbkup.1" ]; then
				./installman.sh >phonebook.log 2>&1
			fi
		fi
		./phonebook -N ${DBNAME} >pbconsole.out 2>&1 &
		# if [ "${DEVTESTING}" -ne "1" ]; then
		# 	if [ ${IAM} == "root" ]; then
		# 		/bin/su - ec2-user -c "~ec2-user/apps/phonebook/pbwatchdog >pbwatchdogstartup.out 2>&1" &
		# 	else
				./pbwatchdog >pbwatchdogstartup.out 2>&1 &
		# 	fi
		# fi
		echo "OK"
		exit 0
		;;
	"stop")
		#===============================================
		# STOP
		# Add the command to terminate your application...
		#===============================================
		stopwatchdog
		curl -s http://${HOST}:${PORT}/extAdminShutdown/
		echo "OK"
		exit 0
		;;
	"ready")
		#===============================================
		# READY
		# Is your application ready to receive commands?
		#===============================================
		ST=$(curl -s http://${HOST}:${PORT}/status/)
		echo "${ST}"
		exit 0
		;;
	"restart")
		#===============================================
		# RESTART
		# Restart your application...
		#===============================================
cat >x.sh << ZZEOF1
curl -s http://${HOST}:${PORT}/extAdminShutdown/ 
echo "sleeping 10 seconds before restart..."
sleep 10
echo "starting phonebook"
./phonebook  -N ${DBNAME} -B ${DBUSER} >phonebook.log 2>&1 &
ZZEOF1
		chmod +x x.sh
		./x.sh >x.sh.log 2>&1 &
		echo "OK"
		exit 0
		;;
	*)
		echo "Unrecognized command: $arg"
		usage
		;;
    esac
done
