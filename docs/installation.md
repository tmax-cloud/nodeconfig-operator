# 설치 및 삭제 가이드

## Prerequisites
* git
* go version v1.16+.
* cri-o version 1.21.2+.
* kubectl version v1.21.3+.
* Access to a Kubernetes v1.21.3+ cluster.
* install cert-manager v1.5.3+

### Operator dependency 
* BMO(BareMetal-Operator) - capm3-v0.3.2: (https://github.com/metal3-io/baremetal-operator.git)


## 설치 가이드
### Step 0. Ironic 설정
* 목적 : Ironic 기동 네트워크 환경에 따라 아래 주소 변경 필요
* 아래의 command를 실행하여 사용하고자 네트워크 설정으로 변경
    ```bash
    $ sed -i 's/192\.168\.111\.1\//${TARGET_API_SERVER_IP}/' ${NCO_HOME}/deploy/deploy_ironic.yaml
    $ sed -i 's/192\.168\.111\.1\:/${TARGET_API_SERVER_IP}/' ${BMO_HOME}/deploy/deploy_ironic.yaml
    $ sed -i 's/192\.168\.111\.100\,192\.168\.111\.200/${TARGET_DHCP_RANGE}/' ${BMO_HOME}/deploy/deploy_ironic.yaml
    ```
    
* 비고 :
    * 설치되는 환경에 따라 IRONIC이 사용하는 네트워크 수정을 원할 경우 BareMetal Operator에서 아래 파일을 수정한다.
    - [IRONIC Configuration](https://github.com/metal3-io/baremetal-operator/blob/master/docs/configuration.md)
  ```bash
	# Ironic 구동에 필요한 설정: ${BMO_HOME}/ironic-deployment/default/ironic_bmo_configmap.env
	# BMO와 Ironic간 통신을 위해 필요한 설정: ${BMO_HOME}/deploy/ironic_ci.env
  ```
  
### Step 1. CR 작성
* 설치할 정보를 [nodeconfig CR](https://github.com/tmax-cloud/nodeconfig-operator/blob/master/docs/api.md)에 작성하여 등록한다.


### Step 2. BMO/NC Operator 설치
* 아래의 명령어를 사용하여 BMO와 NCO가 정상적으로 설치되었는지 확인한다.
  ```bash
  $ kubectl create namespace metal3
  $ kubectl apply -f deploy/deploy_ironic.yaml
  $ kubectl apply -f deploy/deploy_bmh.yaml
  $ kubectl apply -f deploy/deploy_nco.yaml
  $ kubectl get pods -n metal3
  ```
* [Step1](https://github.com/tmax-cloud/nodeconfig-operator#step-1-CR-작성)에서 작성한 nodeconfig CR 등록
  ```bash
  $ kustomize -n metal3 apply $nodecofig
  ```
  
### Step 3. 동작 확인
* *ironic-deployment/default/ironic_bmo_configmap.env*에 작성한 서비스 주소 접근 테스트
  ```bash
  $ curl ${IRONIC_ENDPOINT_IPADDR}:6385/v1/
  $ wget -P /tmp ${IRONIC_ENDPOINT_IPADDR}:${HTTP_PORT}/images/ironic-python-agent.initramfs
  ```
* Ironic pod에있는 dnsmasq의 DHCP 서비스 정상 동작 여부 확인
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
