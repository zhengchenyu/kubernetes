# kubebuilder-example

本文使用kubebuilder构造一个简单的operator。

## 1 使用方案

```
# 1 安装必要的工具
## 1.1 安装kubebuilder
curl -L -o kubebuilder "https://go.kubebuilder.io/dl/latest/$(go env GOOS)/$(go env GOARCH)"
chmod +x kubebuilder && mv kubebuilder /usr/local/bin/


# 2 创建资源
## 2.1 初始化
### 初始化后会下载依赖，同时得到初始化目录
kubebuilder init --domain my.domain --license apache2 --owner zhengchenyu
## 2.2 创建API
### 选择是否创建Resource和Controller, 会生成对应的文件
kubebuilder create api --group my.group --version v1alpha1 --kind Echo
kubebuilder create api --group my.group --version v1alpha1 --kind Guestbook

# 3 编译执行
## 3.1 环境准备
### 编译执行之前，先创建对应的ns
kubectl create ns zcytest

## 3.2 本机上编译执行
### 编译和安装
make install
### 安装CRD
kubectl apply -f config/crd/base
### 编译和执行
make run

## 3.3 本机上编译执行
略

# 4 定义api和逻辑
## 4.1 定义API
### 在echo_types.go/guestbook_types.go中修改如下内容:
### (1) 在EchoSpec/GuestbookSpec增加Spec的自定义内容
### (2) 在EchoStatus/GuestbookStatus增加Status的自定义内容

## 4.2 定义逻辑
### 在echo_controller.go和guestbook_controller.go中修改如下内容
### (1) 在EchoReconciler::Reconcile增加具体的逻辑
### (2) 在GuestbookReconciler::Reconcile增加具体的逻辑

## 4.3 重新编译执行
kubectl apply -f config/crd/base
make run
```