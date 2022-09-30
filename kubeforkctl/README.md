# kubeforkctl

This is a command line tool to generate manifests for Fork resources in kubefork.
([Japanese](../docs/kubeforkctl-doc-ja.md))

- [kubeforkctl](#kubeforkctl)
  - [Requirement](#requirement)
  - [Installation](#installation)
  - [Usage](#usage)
  - [NOTE](#note)
    - [Note on selectors in services selected by `--service-label` or `--service-name`](#note-on-selectors-in-services-selected-by---service-label-or---service-name)
    - [Priority of labels chosen by `--service-hoge` and `--deployment-fuga`](#priority-of-labels-chosen-by---service-hoge-and---deployment-fuga)
    - [Precedence of labels chosen by `--hoge-label` and `--fuga-name`](#precedence-of-labels-chosen-by---hoge-label-and---fuga-name)

---

## Requirement

Go 1.16+

## Installation

```sh
$ go install github.com/wantedly/kubefork-controller/kubeforkctl@latest
```

## Usage

Please check with the following command.

```sh
$ kubeforkctl -h
```

## NOTE

### Note on selectors in services selected by `--service-label` or `--service-name`

When using the `--service-label` or `--service-name` options to retrieve containers for kubefork, search for Deployments and Containers for kubefork using the Service selector that matches options. Note that the Service selector is originally used to select the target Pod, but here we select Deployment.

### Priority of labels chosen by `--service-hoge` and `--deployment-fuga`

If both Service and Deployment are specified in the options, the combination of the selector in Service for kubefork and the Deployment selector specified in the options to search for Deployments. At this time, if there are `matchLabels` with the same label keys but different values in the Service's selector and the option for Deployment, the value will take precedence over the option for Deployment.

### Precedence of labels chosen by `--hoge-label` and `--fuga-name`

The `--service-name` and `--deployment-name` options retrieve all matching labels and include them in the selector to allow selection of resources to be kubefroked by name. If the key of the label specified with `--service-label` and the key of the label obtained with `--service-name` are the same, the value of the label will be the one set with `--service-label` (The same for `--deployment-label` and `--deployment-name`).
