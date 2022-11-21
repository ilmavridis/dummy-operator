# A dummy Kubernetes operator

A dummy Kubernetes operator written with the Operator SDK and Go.

## Description
This operator performs two core functions:
1. Creates a custom API type called "Dummy". <br>Each Dummy has:
    - a spec field which contains a string subfield called "message".
    - a status field which contains a string subfield called "specEcho".
    - a status field that tracks the status of the Pod associated with the Dummy called "podStatus".

2. Creates a custom controller for resources of kind Dummy. <br>The controller:
    - logs the name, namespace and message of each Dummy.
    - copies the value of spec.message to status.specEcho.
    - creates a Pod for each Dummy API object created, Pods run nginx.
    - deletes a Pod if its Dummy ceases to exist.





## Deploy the Operator to a Kubernetes cluster
**Note:** You will need an up and running Kubernetes cluster. You can deploy a Kubernetes cluster locally using [minikube](https://minikube.sigs.k8s.io/docs/start/).


1. Clone the repo
    ```sh
    git clone https://github.com/ilmavridis/dummy-operator
    ```

2. Deploy the CRD to the cluster
    ```sh
    make install
    ```

3. Deploy the controller to the cluster
    ```sh
    make deploy IMG=docker.io/mavridis/k8s-dummy-operator:v0.0.1
    ```

4. Deploy a Dummy object eg.
    ```yaml
    cat <<EOF | kubectl create -f -
    apiVersion: interview.com/v1alpha1
    kind: Dummy
    metadata:
        name: dummy1
        namespace: default
    spec:
        message: "I'm just a dummy"
    EOF
    ```

5. Cleanup

    ```sh
    make undeploy uninstall
    ```

### Build and push the image
You can also build and push the image to a different registry:

```sh
make docker-build docker-push IMG=<some-registry>/dummy-operator:tag
```

### Modify the API definitions
If you are edit the API definitions, generate the manifests using:

```sh
make manifests
```






## Test it out

You can implement different scenarios and see how the controller reacts to different conditions. <br>We present 2 different  test scenarios:



## Scenario A 
1. Create a Dummy
1. Delete the associated Pod 
2. Change the associated Pod's image
3. Delete the Dummy


### 1. A user deploys a Dummy
```yaml
cat <<EOF | kubectl create -f -
apiVersion: interview.com/v1alpha1
kind: Dummy
metadata:
    name: dummy1
    namespace: default
spec:
    message: "I'm just a dummy"
EOF
``` 

- Dummy is produced
    ```yaml
    $ kubectl get dummies -o yaml
    apiVersion: v1
    items:
    - apiVersion: interview.com/v1alpha1
    kind: Dummy
    metadata:
        ...
        name: dummy1
        namespace: default
    spec:
        message: I'm just a dummy
    status:
        PodStatus: Running
        specEcho: I'm just a dummy
    kind: List
    ...
    ```



- Pod is produced
    ```yaml
    $ kubectl get pods -o yaml
    apiVersion: v1
    items:
    - apiVersion: v1
    kind: Pod
    metadata:
        creationTimestamp: "2022-10-28T13:25:06Z"
        name: dummy1
        namespace: default
        ownerReferences:
        - apiVersion: interview.com/v1alpha1
        blockOwnerDeletion: true
        controller: true
        kind: Dummy
        name: dummy1
        ...
    spec:
        containers:
        - image: nginx
        imagePullPolicy: Always
        name: nginx
        ...
    status:
        containerStatuses:
        - containerID: containerd://e02dd97e50b0fa548088cdb48730430f845bf0e750d342fea15ceea571126dd4
        image: docker.io/library/nginx:latest
        imageID: docker.io/library/nginx@sha256:943c25b4b66b332184d5ba6bb18234273551593016c0e0ae906bab111548239f
        name: nginx
        ready: true
        ...
    ```

- Logs 
    ```yaml
    1.666964111661215e+09   INFO    A Dummy has been successfully deployed. {"controller": "dummy", "controllerGroup": "interview.com", "controllerKind": "Dummy", "dummy": {"name":"dummy1","namespace":"default"}, "namespace": "default", "name": "dummy1", "reconcileID": "271480dd-5844-42fd-bb9f-c3e721629a2f", "its Pod is": "Pending"}
    1.6669641137664945e+09  INFO    A Dummy has been successfully deployed. {"controller": "dummy", "controllerGroup": "interview.com", "controllerKind": "Dummy", "dummy": {"name":"dummy1","namespace":"default"}, "namespace": "default", "name": "dummy1", "reconcileID": "05c327df-e5ae-4319-b1ee-4f1ce60a73c4", "its Pod is": "Running"}
    1.6669641137666986e+09  INFO    A Dummy and its Pod have been successfully deployed     {"controller": "dummy", "controllerGroup": "interview.com", "controllerKind": "Dummy", "dummy": {"name":"dummy1","namespace":"default"}, "namespace": "default", "name": "dummy1", "reconcileID": "29916162-f8d8-4839-a641-9ca9cf8e8e6c", "name": "dummy1", "namespace": "default", "message": "I'm just a dummy"}
    ```

### 2. A user deletes the Pod
```sh
$ kubectl delete pod dummy1
pod "dummy1" deleted
```
- The controller redeploys the Pod
    ```sh
    $ kubectl get pods -w
    NAME     READY   STATUS             RESTARTS   AGE
    dummy1   0/1     Pending             0          0s
    dummy1   0/1     Pending             0          0s
    dummy1   0/1     ContainerCreating   0          0s
    dummy1   1/1     Running             0          2s
    ## Pod was deleted here
    dummy1   1/1     Terminating         0          75s
    dummy1   0/1     Terminating         0          76s
    dummy1   0/1     Terminating         0          76s
    dummy1   0/1     Terminating         0          76s
    dummy1   0/1     Pending             0          1s
    dummy1   0/1     Pending             0          1s
    dummy1   0/1     ContainerCreating   0          1s
    dummy1   1/1     Running             0          4s
    ```




### 3. A user changes the Pod's image


```sh
$ kubectl patch pod dummy1 -p '{"spec":{"containers":[{"name": "nginx","image": "redis"}]}}'
pod/dummy1 patched
```

- The controller changes the Pod's image to nginx and the Pod restarts
    ```sh
    $ kubectl get pods -w
    NAME     READY   STATUS             RESTARTS   AGE
    dummy1   0/1     Pending             0          1s
    dummy1   0/1     Pending             0          1s
    dummy1   0/1     ContainerCreating   0          1s
    dummy1   1/1     Running             0          4s
    ## Pod image was changed here
    dummy1   1/1     Running             0          76s
    dummy1   1/1     Running             0          76s
    dummy1   1/1     Running             1 (2s ago)   78s
    dummy1   1/1     Running             2 (2s ago)   80s
    ```
 - The Pod's image was changed to nginx by the controller   
    ```sh
    $ kubectl get pod dummy1 -o jsonpath="{..image}"
    nginx docker.io/library/nginx:latest
    ```


- Logs 
    ```yaml
    1.6669642017041702e+09  INFO    Update Pod's image      {"controller": "dummy", "controllerGroup": "interview.com", "controllerKind": "Dummy", "dummy": {"name":"dummy1","namespace":"default"}, "namespace": "default", "name": "dummy1", "reconcileID": "8d5344c2-d098-4fa3-9288-94f85a42d754"}
    1.6669642017094553e+09  INFO    A Dummy and its Pod have been successfully deployed     {"controller": "dummy", "controllerGroup": "interview.com", "controllerKind": "Dummy", "dummy": {"name":"dummy1","namespace":"default"}, "namespace": "default", "name": "dummy1", "reconcileID": "946c5199-3321-4687-bdd8-a59e30321842", "name": "dummy1", "namespace": "default", "message": "I'm just a dummy"}

    ```

### 4. A user deletes the Dummy
```sh
$ kubectl delete dummy dummy1
dummy.interview.com "dummy1" deleted
```
- Pod deleted
    ```sh
    $ kubectl get pods -w
    NAME     READY   STATUS             RESTARTS   AGE
    dummy1   1/1     Running             0          76s
    dummy1   1/1     Running             0          76s
    dummy1   1/1     Running             1 (2s ago)   78s
    dummy1   1/1     Running             2 (2s ago)   80s
    ## Dummy was deleted here
    dummy1   1/1     Terminating         2 (66s ago)   2m24s
    dummy1   0/1     Terminating         2             2m25s
    dummy1   0/1     Terminating         2             2m25s
    dummy1   0/1     Terminating         2             2m25s

    ```

- Logs
    ```yaml
    1.6669642361329741e+09  INFO    Dummy not found, its Pod is deleted     {"controller": "dummy", "controllerGroup": "interview.com", "controllerKind": "Dummy", "dummy": {"name":"dummy1","namespace":"default"}, "namespace": "default", "name": "dummy1", "reconcileID": "faa67421-2721-47f8-971a-3118b5e147df"}
    1.6669642371620522e+09  INFO    Dummy not found, its Pod is deleted     {"controller": "dummy", "controllerGroup": "interview.com", "controllerKind": "Dummy", "dummy": {"name":"dummy1","namespace":"default"}, "namespace": "default", "name": "dummy1", "reconcileID": "0316a9e7-a92f-40e9-92a4-690f4df963b1"}
    ```






---


## Scenario B 
In this case the Pod already exists but with a different image
1. Create a Pod
1. Create a Dummy
1. Delete the Pod 
2. Change Pod's image
3. Delete Dummy


### 1. Suppose the Pod already exists but its image is redis and not nginx 

A user or a process deploys the Pod

```yaml
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: dummy1
spec:
  containers:
  - name: nginx
    image: redis
EOF
```
```sh
kubectl get pod dummy1 -o jsonpath="{..image}"
redis docker.io/library/redis:latest
```

### 2. User deploys a Dummy
```yaml
cat <<EOF | kubectl create -f -
apiVersion: interview.com/v1alpha1
kind: Dummy
metadata:
    name: dummy1
    namespace: default
spec:
    message: "I'm just a dummy"
EOF
```




- The controller changes the Pod's image to nginx and the Pod is restarted
    ```sh
    $ kubectl get pods -w
    NAME     READY   STATUS             RESTARTS   AGE
    dummy1   0/1     Pending             0          0s
    dummy1   0/1     Pending             0          0s
    dummy1   0/1     ContainerCreating   0          0s
    dummy1   1/1     Running             0          2s
    ## Dummy was created here
    dummy1   1/1     Running             0          34s
    dummy1   1/1     Running             1 (3s ago)   37s
    ```

 - The Pod's image was changed to nginx by the controller 
    ```sh
    $ kubectl get pod dummy1 -o jsonpath="{..image}"
    nginx docker.io/library/nginx:latest
    ```
- A finalizer was added to the Dummy to implement cleanup logic because there is no relationship between the Dummy and its Pod.
    ```sh
    $ kubectl get dummy -o yaml | grep -i finalizer
    finalizers:
    - interview.com/finalizer
    ```

- Logs
    ```yaml
    1.669033144666449e+09   INFO    There is no relationship established between the Dummy and its Pod      {"controller": "dummy", "controllerGroup": "interview.com", "controllerKind": "Dummy", "dummy": {"name":"dummy1","namespace":"default"}, "namespace": "default", "name": "dummy1", "reconcileID": "42092074-d2cc-4cd8-bd80-771285267bab"}
    1.6690331446719627e+09  INFO    Added finalizer {"controller": "dummy", "controllerGroup": "interview.com", "controllerKind": "Dummy", "dummy": {"name":"dummy1","namespace":"default"}, "namespace": "default", "name": "dummy1", "reconcileID": "42092074-d2cc-4cd8-bd80-771285267bab"}
    1.6690331446768754e+09  INFO    Update Pod's image      {"controller": "dummy", "controllerGroup": "interview.com", "controllerKind": "Dummy", "dummy": {"name":"dummy1","namespace":"default"}, "namespace": "default", "name": "dummy1", "reconcileID": "42092074-d2cc-4cd8-bd80-771285267bab"}
    1.6690331446833532e+09  INFO    A Dummy and its Pod have been successfully deployed     {"controller": "dummy", "controllerGroup": "interview.com", "controllerKind": "Dummy", "dummy": {"name":"dummy1","namespace":"default"}, "namespace": "default", "name": "dummy1", "reconcileID": "87bfd552-526c-4ba0-a604-393bace82395", "name": "dummy1", "namespace": "default", "message": "I'm just a dummy"}
    ```

#### 4. A user deletes the Dummy
```sh
$ kubectl delete dummy dummy1
dummy.interview.com "dummy1" deleted
```
- Pod deleted
    ```sh
    $ kubectl get pods -w
    NAME     READY   STATUS             RESTARTS   AGE
    dummy1   1/1     Running             0          34s
    dummy1   1/1     Running             1 (3s ago)   37s
    ## Dummy was deleted here
    dummy1   1/1     Terminating         1 (78s ago)   112s
    dummy1   0/1     Terminating         1 (79s ago)   113s
    dummy1   0/1     Terminating         1 (79s ago)   113s
    dummy1   0/1     Terminating         1 (79s ago)   113s

    ```

