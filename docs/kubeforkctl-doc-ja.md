# kubeforkctl

kubeforkにおけるForkリソースのマニフェストを生成するコマンドラインツールです。

- [kubeforkctl](#kubeforkctl)
  - [Requirement](#requirement)
  - [Installation](#installation)
  - [Usage](#usage)
  - [NOTE](#note)
    - [`--service-label`または`--service-name`によって選択するService中のセレクタの注意点](#--service-labelまたは--service-nameによって選択するservice中のセレクタの注意点)
    - [`--service-hoge`と`--deployment-fuga`によって選ばれるラベルの優先度](#--service-hogeと--deployment-fugaによって選ばれるラベルの優先度)
    - [`--hoge-label`と`--fuga-name`によって選ばれるラベルの優先度](#--hoge-labelと--fuga-nameによって選ばれるラベルの優先度)

---

## Requirement

Go 1.16+

## Installation

```sh
$ go install github.com/wantedly/kubefork-controller/kubeforkctl@latest
```

## Usage

以下のコマンドで確認してください。

```sh
$ kubeforkctl -h
```

## NOTE

### `--service-label`または`--service-name`によって選択するService中のセレクタの注意点

`--service-label`または`--service-name`オプションを用いて、kubeforkの対象となるコンテナを取得する際、その条件に合致するServiceのセレクタを用いて、kubeforkの対象となるDeploymentおよびコンテナを検索します。Serviceのセレクタは本来対象となるPodを選ぶためのものですが、ここではDeploymentを選んでいることに注意してください。

### `--service-hoge`と`--deployment-fuga`によって選ばれるラベルの優先度

オプションでServiceとDeploymentの両方を指定している場合、kubeforkの対象となるServiceのセレクタと、オプションで指定されたDeploymentを選ぶためのセレクタを統合してkubeforkの対象となるDeploymentを検索します。この時、Serviceのセレクタとオプションで指定したDeploymentの指定に、ラベルキーが等しく値の異なる`matchLabels`が存在した場合、その値はオプションで指定したDeploymentの指定の方が優先されます。

### `--hoge-label`と`--fuga-name`によって選ばれるラベルの優先度

`--service-name`や`--deployment-name`といったオプションは、合致する名前のService・Deploymentのすべてのラベルを取得し、その情報をセレクタに入れることでkubefrokするリソースの名前による選択を可能にしています。この時、`--service-label`で指定したラベルのキーと`--service-name`で取得したラベルのキーが同じ場合、そのラベルの値は`--service-label`で設定した方になります（`--deployment-label`と`--deployment-name`についても同様）。
