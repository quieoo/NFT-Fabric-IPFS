docker kill logspout
sleep 3
echo "==shuting down network=="
./network.sh down
echo ""
echo "==shut down successfully=="