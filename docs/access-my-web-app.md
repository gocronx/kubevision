# 通过 KubeVision 访问 my-web-app

## 当前状态

- Deployment: `my-web-app`（2 副本，nginx:latest，容器端口 80）
- Service: **未创建** — 需要先创建 Service 才能访问

## 步骤

### 1. 创建 Service

1. 在左侧菜单进入 **Services** 页面
2. 点击右上角 **Create** 按钮
3. 在弹窗中选择模板 **ClusterIP Service**（或手动填写以下 JSON）：

```json
{
  "apiVersion": "v1",
  "kind": "Service",
  "metadata": {
    "name": "my-web-app",
    "namespace": "default"
  },
  "spec": {
    "selector": {
      "app": "my-web-app"
    },
    "ports": [
      {
        "port": 80,
        "targetPort": 80,
        "protocol": "TCP"
      }
    ],
    "type": "ClusterIP"
  }
}
```

4. 点击 **Create** 提交

### 2. 浏览器访问

Service 创建后，有两种方式访问：

#### 方式 A：kubectl port-forward（本地开发推荐）

```bash
kubectl port-forward svc/my-web-app 8080:80 -n default
```

然后浏览器打开：**http://localhost:8080**

#### 方式 B：改为 NodePort 类型

如果需要集群外直接访问，创建 Service 时把 `type` 改为 `NodePort`：

```json
{
  "spec": {
    "type": "NodePort",
    "selector": {
      "app": "my-web-app"
    },
    "ports": [
      {
        "port": 80,
        "targetPort": 80,
        "nodePort": 30080
      }
    ]
  }
}
```

然后浏览器打开：**http://<节点IP>:30080**

#### 方式 C：LoadBalancer（云环境）

把 `type` 改为 `LoadBalancer`，云厂商会自动分配外部 IP。

### 3. 验证

访问后应该看到 **Welcome to nginx!** 默认页面，说明部署成功。
