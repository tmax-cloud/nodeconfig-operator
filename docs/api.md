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

* *goodCredentials* -- A reference to the secret and its namespace
  holding the last set of BMC credentials the system was able to validate
  as working.

* *triedCredentials* -- A reference to the secret and its namespace
  holding the last set of BMC credentials that were sent to
  the provisioning backend.

* *lastUpdated* -- The timestamp of the last time the status of the
  host was updated.

* *operationalStatus* -- The status of the server. Value is one of the
  following:
  * *OK* -- Indicates all the details for the host are known and working,
    meaning the host is correctly configured and manageable.
  * *discovered* -- Implies some of the host's details are either
    not working correctly or missing. For example, the BMC address is known
    but the login credentials are not.
  * *error* -- Indicates the system found some sort of irrecuperable error.
    Refer to the *errorMessage* field in the status section for more details.

* *errorMessage* -- Details of the last error reported by the provisioning
  backend, if any.

* *hardware* -- The details for hardware capabilities discovered on the
  host. These are filled in by the provisioning agent when the host is
  registered.
  * *nics* -- List of network interfaces for the host.
    * *name* -- A string identifying the network device,
      e.g. *nic-1*.
    * *mac* -- The MAC address of the NIC.
    * *ip* -- The IP address of the NIC, if one was assigned
      when the discovery agent ran.
    * *speedGbps* -- The speed of the device in Gbps.
    * *vlans* -- A list holding all the VLANs available for this NIC.
    * *vlanId* -- The untagged VLAN ID.
    * *pxe* -- Whether the NIC is able to boot using PXE.
  * *storage* -- List of storage (disk, SSD, etc.) available to the host.
    * *name* -- A string identifying the storage device,
      e.g. *disk 1 (boot)*.
    * *rotational* -- Either true or false, indicates whether the disk
      is rotational.
    * *sizeBytes* -- Size of the storage device.
    * *serialNumber* -- The device's serial number.
  * *cpu* -- Details of the CPU(s) in the system.
    * *arch* -- The architecture of the CPU.
    * *model* -- The model string.
    * *clockMegahertz* -- The speed in GHz of the CPU.
    * *flags* -- List of CPU flags, e.g. 'mmx','sse','sse2','vmx', ...
    * *count* -- Amount of these CPUs available in the system.
  * *firmware* -- Contains BIOS information like for instance its *vendor*
    and *version*.
  * *systemVendor* -- Contains information about the host's *manufacturer*,
    the *productName* and *serialNumber*.
  * *ramMebibytes* -- The host's amount of memory in Mebibytes.

* *hardwareProfile* -- **This field is deprecated. See rootDeviceHints instead.**
  The name of the hardware profile that matches the
  hardware discovered on the host based on the details saved
  to the *Hardware* section. If the hardware does not match any
  known profile, the value `unknown` will be set on this field
  and is used by default. In practice, this only affects which device
  the OS image will be written to. The following are the current
  supported `hardwareProfile` settings and their corresponding root devices.

  | **hardwareProfile** | **Root Device** |
  |---------------------|-----------------|
  | `unknown`           | /dev/sda        |
  | `libvirt`           | /dev/vda        |
  | `dell`              | HCTL: 0:0:0:0   |
  | `dell-raid`         | HCTL: 0:2:0:0   |
  | `openstack`         | /dev/vdb        |

  **NOTE:** These are subject to change.

* *poweredOn* -- Boolean indicating whether the host is powered on.
  See *online* on the *BareMetalHost's* *Spec*.

* *provisioning* -- Settings related to deploying an image to the host.
  * *state* -- The current state of any ongoing provisioning operation.
    The following are the currently supported ones:
    * *\<empty string\>* -- There is no provisioning happening, at the moment.
    * *registration error* -- The details for the host's BMC are
      either incorrect or incomplete therfore the host could not be managed.
    * *registering* -- The host's BMC details are being checked.
    * *match profile* -- The discovered hardware details on the host
      are being compared against known profiles.
    * *ready* -- The host is available to be consumed.
    * *provisioning* -- An image is being written to the host's disk(s).
    * *provisioning error* -- The image could not be written to the host.
    * *provisioned* -- An image has been completely written to the host's
      disk(s).
    * *externally provisioned* -- Metal³ does not manage the image on the host.
    * *deprovisioning* -- The image is being wiped from the host's disk(s).
    * *inspecting* -- The hardware details for the host are being collected
      by an agent.
    * *power management error* -- An error was found while trying to power
      the host either on or off.
  * *id* -- The unique identifier for the service in the underlying
    provisioning tool.
  * *image* -- The image most recently provisioned to the host.
  * *rootDeviceHints* -- The root device selection instructions used
    for the most recent provisioning operation.

### BareMetalHost Example

The following is a complete example from a running cluster of a *BareMetalHost*
resource (in YAML), it includes its specification and status sections:

```yaml
apiVersion: metal3.io/v1alpha1
kind: BareMetalHost
metadata:
  creationTimestamp: "2019-09-20T06:33:35Z"
  finalizers:
  - baremetalhost.metal3.io
  generation: 2
  name: bmo-master-0
  namespace: bmo-project
  resourceVersion: "22642"
  selfLink: /apis/metal3.io/v1alpha1/namespaces/bmo-project/baremetalhosts/bmo-master-0
  uid: 92b2f77a-db70-11e9-9db1-525400764849
spec:
  bmc:
    address: ipmi://10.10.57.19
    credentialsName: bmo-master-0-bmc-secret
  bootMACAddress: 98:03:9b:61:80:48
  consumerRef:
    apiVersion: machine.openshift.io/v1beta1
    kind: Machine
    name: bmo-master-0
    namespace: bmo-project
  externallyProvisioned: true
  hardwareProfile: default
  image:
    checksum: http://172.16.1.100/images/myOSv1/myOS.qcow2.md5sum
    url: http://172.16.1.100/images/myOSv1/myOS.qcow2
  online: true
  userData:
    name: bmo-master-user-data
    namespace: bmo-project
  networkData:
    name: bmo-master-network-data
    namespace: bmo-project
  metaData:
    name: bmo-master-meta-data
    namespace: bmo-project
status:
  errorMessage: ""
  goodCredentials:
    credentials:
      name: bmo-master-0-bmc-secret
      namespace: bmo-project
    credentialsVersion: "5562"
  hardware:
    cpu:
      arch: x86_64
      clockMegahertz: 2000
      count: 40
      flags: []
      model: Intel(R) Xeon(R) Gold 6138 CPU @ 2.00GHz
    firmware:
      bios:
        date: 12/17/2018
        vendor: Dell Inc.
        version: 1.6.13
    hostname: bmo-master-0.localdomain
    nics:
    - ip: 172.22.135.105
      mac: "00:00:00:00:00:00"
      model: unknown
      name: eno1
      pxe: true
      speedGbps: 25
      vlanId: 0
    ramMebibytes: 0
    storage: []
    systemVendor:
      manufacturer: Dell Inc.
      productName: PowerEdge r460
      serialNumber: ""
  hardwareProfile: ""
  lastUpdated: "2019-09-20T07:03:23Z"
  operationalStatus: OK
  poweredOn: true
  provisioning:
    ID: a4438010-3fc6-4c5c-b570-900bbe85da57
    image:
      checksum: ""
      url: ""
    state: externally provisioned
  triedCredentials:
    credentials:
      name: bmo-master-0-bmc-secret
      namespace: bmo-project
    credentialsVersion: "5562"
```

And here it is the secret `bmo-master-0-bmc-secret` holding the host's BMC credentials:

```yaml
---
apiVersion: v1
kind: Secret
metadata:
  name: bmo-master-0-bmc-secret
type: Opaque
data:
  username: YWRtaW4=
  password: cGFzc3dvcmQ=
```

## Triggering Provisioning

Several conditions must be met in order to initiate provisioning.

1. The host `spec.image.url` field must contain a URL for a valid
   image file that is visible from within the cluster and from the
   host receiving the image.
2. The host must have `online` set to `true` so that the operator will
   keep the host powered on.

To initiate deprovisioning, clear the image URL from the host spec.
