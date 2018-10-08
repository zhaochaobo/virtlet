# 总览

由于虚拟机需要独立网络访问，故提出桥接网络方案。桥接网络可以直接打通与物理网络的连接，直接支持dhcp，静态ip配置等功能。 缺点是无法为桥接网络定制安全策略。

# 环境要求

由于桥接网络无法引用网络安全策略，故要求需要kvm调度的节点机器有2张以上的网卡，一张用于原生的k8s pod网络。 另外一张用于虚拟机的桥接网络，与物理网络直接打通。
* 执行命令将节点上用于虚拟机网络的网卡加入到网桥br0中。

```
nmcli con add ifname br0 type bridge con-name br0
nmcli con add type bridge-slave ifname ens15f0  master br0
```

# 为kvm创建bridge cni配置文件，注意不要和原有k8s的cni配置目录冲突了。注意目录名是net1.d
， 网桥名字br0要和上面步骤的名称一致。

```

[root@node6 ~]# cat /etc/cni/net1.d/9-bridge.conf
{
        "name": "mynet",
        "type": "bridge",
        "bridge": "br0",
        "isDefaultGateway": true,
        "forceAddress": false,
        "ipMasq": true,
        "hairpinMode": true,
        "ipam": {
                "type": "dhcp"
        }
}

```

# 在节点上运行dhcp中继守护进程。让虚拟机启动时拿到dhcp地址。 注意如果提示有dhcp.sock已经存在，需要将文件删除。

```
[root@node6 ~]# /opt/cni/bin/dhcp daemon
```

# 配置virtlet服务使用bridge网络来创建虚拟机。

* 配置configmap，使用virtlet_cni_config_dir配置bridge的cni目录。

```

apiVersion: v1
data:
  calico-subnet: "22"
  download_protocol: http
  image_regexp_translation: "1"
  loglevel: "9"
  virtlet_cni_config_dir: /etc/cni/net1.d
kind: ConfigMap
metadata:
  name: virtlet-config
  namespace: kube-system

```

* 修改配置virtlet daemonset，使用bridge网络来添加虚拟机网络。注意只添加VIRTLET_CNI_CONFIG_DIR 段。

```
      initContainers:
      - command:
        - /prepare-node.sh
        env:
        - name: KUBE_NODE_NAME
          valueFrom:
            fieldRef:
              apiVersion: v1
              fieldPath: spec.nodeName
        - name: VIRTLET_CNI_CONFIG_DIR
          valueFrom:
            configMapKeyRef:
              key: virtlet_cni_config_dir
              name: virtlet-config
              optional: true
```

# 创建windows虚拟机, 注意虚拟机的ip是要在windows配置了virtio网卡驱动后, 才能正确发出dhcp请求，拿到dhcp地址.（windows虚拟机第一次运行需要vnc登陆配置。

```
[root@master1 ~]# kubectl get pod -o wide
NAME                                   READY     STATUS    RESTARTS   AGE       IP               NODE
calico-policy-controller-dw4pk         1/1       Running   0          10d       192.168.6.5      node5
kube-apiserver-master1                 1/1       Running   0          10d       192.168.6.1      master1
kube-controller-manager-master1        1/1       Running   2          11d       192.168.6.1      master1
kube-dns-6bb4b59877-pmjwc              3/3       Running   6          11d       10.233.97.131    node5
kube-dns-6bb4b59877-qkghj              3/3       Running   0          10d       10.233.104.68    master1
kube-dns-autoscaler-7bd96594fb-9txxd   1/1       Running   0          10d       10.233.74.79     node4
kube-proxy-master1                     1/1       Running   1          11d       192.168.6.1      master1
kube-proxy-node2                       1/1       Running   2          11d       192.168.6.2      node2
kube-proxy-node3                       1/1       Running   2          10d       192.168.6.3      node3
kube-proxy-node4                       1/1       Running   2          10d       192.168.6.4      node4
kube-proxy-node5                       1/1       Running   2          11d       192.168.6.5      node5
kube-proxy-node6                       1/1       Running   5          8d        192.168.22.222   node6
kube-scheduler-master1                 1/1       Running   2          11d       192.168.6.1      master1
virtlet-d8k7z                          3/3       Running   0          4h        192.168.22.222   node6
windowsxp2-vm                          1/1       Running   0          4h        192.168.0.10     node6

```
