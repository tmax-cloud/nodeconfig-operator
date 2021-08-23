# NodeConfig-Operator
## 구성 요소 및 버전
* BMO(BareMetal-Operator) - capm3-v0.3.2: (https://github.com/metal3-io/baremetal-operator.git)

## Prerequisites
* git
* kustomize
* go version v1.16+.
* cri-o version 1.21.2+.
* kubectl version v1.21.3+.
* Access to a Kubernetes v1.21.3+ cluster.

## 설치 가이드
### Step 0. BMO config 수정
* 목적 : BMO version에 맞게 이미지 버전 수정
* 컨테이너 이미지 수정 : 아래의 command를 실행하여 사용하고자 하는 이미지 버전을 수정한다.
    ```bash
    $ git clone -b capm3-v0.3.2 https://github.com/metal3-io/baremetal-operator.git && BMO_HOME=$PWD/baremetal-operator
    $ sed -i 's/quay.io\/metal3-io\/baremetal-operator/quay.io\/metal3-io\/baremetal-operator\:capm3-v0.3.2/' ${BMO_HOME}/deploy/operator/bmo.yaml
    $ sed -i 's/quay.io\/metal3-io\/ironic-inspector/quay.io\/metal3-io\/ironic-inspector\:capm3-v0.3.2/' ${BMO_HOME}/ironic-deployment/ironic/ironic.yaml
    $ sed -i 's/quay.io\/metal3-io\/ironic-ipa-downloader/quay.io\/metal3-io\/ironic-ipa-downloader\:capm3-v0.3.2/' ${BMO_HOME}/ironic-deployment/ironic/ironic.yaml
    $ sed -i 's/quay.io\/metal3-io\/ironic/quay.io\/metal3-io\/ironic\:capm3-v0.3.2/' ${BMO_HOME}/ironic-deployment/ironic/ironic.yaml
    ```
    
* 비고 :
    * 설치되는 환경에 따라 IRONIC이 사용하는 네트워크 수정을 원할 경우 아래 파일을 수정한다.
  ```bash
	# Ironic 구동에 필요한 설정: ${BMO_HOME}/ironic-deployment/default/ironic_bmo_configmap.env
	# BMO와 Ironic간 통신을 위해 필요한 설정: ${BMO_HOME}/deploy/ironic_ci.env
  ```
  
### Step 1. CRD 작성
* 설치할 정보를 기록한 nodeconfig CRD를 작성하여 등록한다.
* [CRD 작성 참고](https://github.com/tmax-cloud/nodeconfig-operator/blob/master/docs/api.md)


### Step 2. BMO/NC-operator 설치
* 아래의 명령어를 사용하여 BMO와 NCO가 정상적으로 설치되었는지 확인한다.
  ```bash
  $ kustomize build ${BMO_HOME}/ironic-deployment/default/ |kubectl apply -f -
  $ kustomize build ${BMO_HOME}/deploy/default/ |kubectl apply -f -
  $ kustomize build ${NCO_HOME}/config/default |kubectl apply -f -
  $ kubectl get pods -n metal3
   ```

### Step 3. 동작 확인
* *ironic-deployment/default/ironic_bmo_configmap.env*에 작성한 서비스 주소 접근 테스트
  ```bash
  $ curl ${IRONIC_ENDPOINT_IPADDR}:6385/v1/
  $ wget -P /tmp ${IRONIC_ENDPOINT_IPADDR}:${HTTP_PORT}/images/ironic-python-agent.initramfs
  ```
* Ironic pod의 dnsmasq의 DHCP 서비스 정상 동작 여부 확인
  ```bash
  $ nmap --script broadcast-dhcp-discover -e ${PROVISIONING_INTERFACE}
  ```

## 삭제 가이드
### Step 1. 사용중인 리소스 제거
  ```bash
  $ kubectl -n metal3 delete nodeconfig --all
  $ kubectl -n metal3 delete bmh --all
  ```
### Step 2. 설치 제거
  ```bash
  $ kubectl -n metal3 delete deploy nodeconfig-controller-manager
  $ kubectl -n metal3 delete deploy metal3-baremetal-operator
  $ kubectl -n metal3 delete deploy metal3-ironic
  $ kubectl delete namespace metal3
  ```

## Resources
* [API documentation](docs/api.md)
* [BMO API documentation](https://github.com/metal3-io/baremetal-operator/blob/capm3-v0.3.2/docs/api.md)
* [BMO Configuration](https://github.com/metal3-io/baremetal-operator/blob/capm3-v0.3.2/docs/configuration.md)
