if [ "$EUID" -ne 0 ]
  then echo "You are not root. Will exit"
  exit
fi

sysctl net.ipv4.conf.all.forwarding=1
iptables -P FORWARD ACCEPT
iptables -t nat -A POSTROUTING -o gocker0 -j MASQUERADE
# Change the name of the interface, ens33 to match that of the
# main interface Ethernet/Wifi you use to connect to the internet
# for routing to work successfully.
iptables -t nat -A POSTROUTING -o ens33 -j MASQUERADE
