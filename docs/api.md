# API and Resource Definitions

## NodeConfig
Main responsibility of NCO(NodeConfig Operator) is to convert a **NodeConfig** bootstrap object into a cloud-init script that is going to turn a Baremetal Machine into a Legacy Node.

The cloud-init script will be saved into the NodeConfig.Status.BootstrapData and then the BMO(BareMetal Operator) will pick up this value and proceed with the machine creation and the actual bootstrap.

NCO also supports BMH(BareMetalHost) resource creation. 
The `NodeConfig.Specs.BMC` is a connection information for the BMC (Baseboard Management 
Controller) on the host and `NodeConfig.Specs.image` contains details for the image to be 
deployed on a given host. Above two sections are used to create **BareMetalHost** resource,
which defines a physical host and its properties. The other **NodeConfig** objects supports 
customizing the content of the config-data

### NodeConfig spec

The *NodeConfig's* *spec* defines the desire state of the host. It contains
mainly, but not only, provisioning details.

#### Spec fields

* *bmc* -- The connection information for the BMC (Baseboard Management
  Controller) on the host.
  * *address* -- The URL for communicating with the BMC controller, based
    on the provider being used, the URL will look as follows:
    * IPMI
      * `ipmi://<host>:<port>`, an unadorned `<host>:<port>` is also accepted
        and the port is optional, if using the default one (623).
  * *bootMode* -- The method of initializing the hardware during boot
  * *username* -- the username for the BMC
  * *password* -- the password for the BMC

* *image* -- Holds details for the image to be deployed on a given host.
  * *url* -- The URL of an image to deploy to the host.
  * *checksum* -- The actual checksum or a URL to a file containing
    the checksum for the image at *image.url*.
  * *checksumType* -- Checksum algorithms can be specified. Currently
    only `md5`, `sha256`, `sha512` are recognized. If nothing is specified
    `md5` is assumed.
    
* *files* -- specifies additional files to be created on the machine
* *cloudInitCommands* -- specifies a list of commands to be executed on first boot(after OS installation)
* *users* -- specifies a list of users to be created on the machine
* *ntp* -- specifies NTP settings for the machine

### NodeConfig status

Moving onto the next block, the *BareMetalHost's* *status* which represents
the host's current state. Including tested credentials, current hardware
details, etc.

#### Status fields
* *ready* -- indicates the BootstrapData field is ready to be consumed
* *dataSecretName* -- the name of the secret that stores the bootstrap data script
* *userData* -- a references the Secret that holds user data needed by the bare metal operator

### NodeConfig Example

The following is a complete example from a running cluster of a *NodeConfig*
resource (in YAML), it includes its specification and status sections:

```yaml
apiVersion: bootstrap.tmax.io/v1alpha1
kind: NodeConfig
metadata:
  name: #Node_ID
  annotations:
    metal3.io/BareMetalHost: "metal3/#Node_ID"
spec:
  bmc:
    address: #IP_ADDR
    bootMode: UEFI
    username: #BMC_USER
    password: #BMC_PWD
  image:
    url: #QCOW2_URL
    checksum: #IMG_CHKSUM_URL
    checksumType: md5
  files:
  - content: |
      TYPE=Ethernet
      BOOTPROTO=static
      DEVICE=eth2
      ONBOOT=yes
      USERCTL=no
      IPADDR=#NODE_IP
      NETMASK=255.255.255.0
      GATEWAY=#GW_IP
    owner: root:root
    path: /etc/sysconfig/network-scripts/ifcfg-eth2
    permissions: "0755"
  cloudInitCommands:
  - adduser tmax
  - ( echo 'root'; echo 'root'; ) | passwd tmax
  - ( echo 'tmax@23'; echo 'tmax@23'; ) | passwd root
  - ifup eth2
  - yum update -y
  - setenforce 0
  - sed -i 's/^SELINUX=enforcing$/SELINUX=permissive/' /etc/selinux/config
  - >-
    yum install gcc kernel-headers kernel-devel keepalived
    device-mapper-persistent-data lvm2
  - systemctl enable --now keepalived
  - sysctl net.bridge.bridge-nf-call-iptables=1
  users:
  - name: #USER_ID
    sshAuthorizedKeys:
    - ssh-rsa XXX
    sudo: ALL=(ALL) NOPASSWD:ALL
```

## Triggering Provisioning

Several conditions must be met in order to initiate provisioning.

1. The BMH `spec.image.url` field must contain a URL for a valid
   image file that is visible from within the cluster and from the
   host receiving the image.
2. The BMH must have `online` set to `true` so that the operator will
   keep the host powered on.

To initiate deprovisioning, clear the image URL from the host spec.
