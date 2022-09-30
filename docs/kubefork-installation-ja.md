# kubeforkの導入方法

このドキュメントでは、kubeforkをクラスタへ導入する方法について説明します。

- [kubeforkの導入方法](#kubeforkの導入方法)
  - [Requirement](#requirement)
  - [kubefork-controllerの導入](#kubefork-controllerの導入)
    - [ForkManagerの導入](#forkmanagerの導入)
  - [kubeforkctlのインストール](#kubeforkctlのインストール)
  - [References](#references)

---

## Requirement

Go 1.16+
kubernetes 1.25+

また、あらかじめ以下をクラスタに導入しておいてください。
[deployment-duplicator](https://github.com/wantedly/deployment-duplicator)
[Istio 1.15+](https://istio.io/latest/docs/setup/getting-started/)
[Emissary-ingress 3.1+](https://www.getambassador.io/docs/emissary/latest/tutorials/getting-started/)

## kubefork-controllerの導入

[kubefork-controller](https://github.com/wantedly/kubefork-controller)を参照してください。

### ForkManagerの導入

kubeforkを行うためには、どのヘッダを見て通信するマイクロサービスを振り分けるか、どのホストとどのサービスが対応するかを設定するForkManagerリソースをapplyする必要があります。もっとも基本的なForkManagerのマニフェストは以下になります。

```yaml
apiVersion: fork.k8s.wantedly.com/v1beta1
kind: ForkManager
metadata:
  name: kubefork-manager
  namespace: manager
spec:
  ambassadorID: fork-ambassador
  headerKey: x-fork-identifier
  upstreams:
  - host: example.com
    original: your-service:80
```

この場合、通信を振り分けるために参照されるヘッダのキーは`x-fork-identifier`となり、また`example.com`への通信は通常時にはサービス`your-service`の80番ポートへ送信されます。

ForkManagerの詳しい仕様については[forkmanager_types.go](https://github.com/wantedly/kubefork-controller/blob/main/api/v1beta1/forkmanager_types.go)を参照してください。

## kubeforkctlのインストール

ForkManagerをapplyした後は、実際にkubeforkを行うためのコマンドラインツールである[kubeforkctl](https://github.com/wantedly/kubefork-controller/tree/main/kubeforkctl)をインストールします。

## References

現在、kubeforkを行うためには、**ユーザは特定のヘッダを伝播させる機能をアプリケーションに搭載する必要があります**。詳しくは[kubeforkのドキュメント](./kubefork-doc-ja.md#どのようにしてkubeforkを行うか)を参照してください。
