# NodeConfig-Operator
## 구성 요소 및 버전
* BMO(BareMetal-Operator) - capm3-v0.3.2: (https://github.com/metal3-io/baremetal-operator.git)

## Prerequisites
* git
* go version v1.16+.
* cri-o version 1.21.2+.
* kubectl version v1.21.3+.
* Access to a Kubernetes v1.21.3+ cluster.

## 설치 가이드
### Step 0. BMO config 수정
* 목적 : `BMO version에 맞게 이미지 버전 수정`
* 생성 순서 : 아래의 command를 실행하여 사용하고자 하는 image 버전을 수정한다.
    ```bash
    $ git clone -b capm3-v0.3.2 https://github.com/metal3-io/baremetal-operator.git && BMO_HOME=$PWD/baremetal-operator
    $ sed -i 's/quay.io\/metal3-io\/baremetal-operator/quay.io\/metal3-io\/baremetal-operator\:capm3-v0.3.2/' ${BMO_HOME}/deploy/operator/bmo.yaml
    $ sed -i 's/quay.io\/metal3-io\/ironic-inspector/quay.io\/metal3-io\/ironic-inspector\:capm3-v0.3.2/' ${BMO_HOME}/ironic-deployment/ironic/ironic.yaml
    $ sed -i 's/quay.io\/metal3-io\/ironic-ipa-downloader/quay.io\/metal3-io\/ironic-ipa-downloader\:capm3-v0.3.2/' ${BMO_HOME}/ironic-deployment/ironic/ironic.yaml
    $ sed -i 's/quay.io\/metal3-io\/ironic/quay.io\/metal3-io\/ironic\:capm3-v0.3.2/' ${BMO_HOME}/ironic-deployment/ironic/ironic.yaml
    ```
    
* 비고 :
    * 설치되는 환경에 따라 IRONIC이 사용하는 Network 수정을 원할 경우 아래 파일을 수정한다.
  ```bash
	# Ironic 구동에 필요한 설정: ${BMO_HOME}/ironic-deployment/default/ironic_bmo_configmap.env
	# BMO와 Ironic간 통신을 위해 필요한 설정: ${BMO_HOME}/deploy/ironic_ci.env
  ```
  
### Step 1. CRDs 생성
* 목적 : `설치할 정보를 기록한 CRD를 배포한다.`


### Step 2. BMO/NC-operator 설치
* 아래의 명령어를 사용하여 BMO와 NCO가 정상적으로 설치되었는지 확인한다.
  ```bash
    $ kubectl get pods -n metal3
   ```


### Step 3. 동작 확인


## 삭제 가이드
### Step 1. 사용중인 리소스 제거
### Step 2. 설치 제거
### Step 3. CRDs 제거


## Resources
* [API documentation](docs/api.md)
* [Setup Development Environment](docs/dev-setup.md)
* [Configuration](docs/configuration.md)
* [Testing](docs/testing.md)
* [Publishing Images](docs/publishing-images.md)
