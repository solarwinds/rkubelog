GO := GO111MODULE=on go

build:
	$(GO) build ./...

test:
	$(GO) test ./...

test-full: example
	$(GO) test -race ./...

test-cover:
	$(GO) test -coverprofile=coverage.txt -covermode=count -coverpkg="./..." $$(go list ./... | grep -v join/gen | grep -v types/gen )
	curl -s https://codecov.io/bash | bash


install-deps:
	$(GO) mod download

generate: generate-types generate-type-tests generate-joins

generate-types:
	genny -in=types/gen/template.go -out=types/pod/generated.go -pkg=pod gen 'ObjectType=*corev1.Pod'
	genny -in=types/gen/template.go -out=types/ingress/generated.go -pkg=ingress gen 'ObjectType=*networkingv1beta1.Ingress'
	genny -in=types/gen/template.go -out=types/secret/generated.go -pkg=secret gen 'ObjectType=*corev1.Secret'
	genny -in=types/gen/template.go -out=types/service/generated.go -pkg=service gen 'ObjectType=*corev1.Service'
	genny -in=types/gen/template.go -out=types/event/generated.go -pkg=event gen 'ObjectType=*corev1.Event'
	genny -in=types/gen/template.go -out=types/node/generated.go -pkg=node gen 'ObjectType=*corev1.Node'
	genny -in=types/gen/template.go -out=types/replicationcontroller/generated.go -pkg=replicationcontroller gen 'ObjectType=*corev1.ReplicationController'
	genny -in=types/gen/template.go -out=types/replicaset/generated.go -pkg=replicaset gen 'ObjectType=*appsv1.ReplicaSet'
	genny -in=types/gen/template.go -out=types/deployment/generated.go -pkg=deployment gen 'ObjectType=*appsv1.Deployment'
	genny -in=types/gen/template.go -out=types/job/generated.go -pkg=job gen 'ObjectType=*batchv1.Job'
	genny -in=types/gen/template.go -out=types/daemonset/generated.go -pkg=daemonset gen 'ObjectType=*appsv1.DaemonSet'
	genny -in=types/gen/template.go -out=types/statefulset/generated.go -pkg=statefulset gen 'ObjectType=*appsv1.StatefulSet'
	goimports -w types/**/generated.go
	$(GO) build ./types/...

generate-type-tests:
	$(GO) build -o ./types/gen/gen ./types/gen
	./types/gen/gen corev1.Pod > types/pod/generated_test.go
	./types/gen/gen networkingv1beta1.Ingress > types/ingress/generated_test.go
	./types/gen/gen corev1.Secret > types/secret/generated_test.go
	./types/gen/gen corev1.Service > types/service/generated_test.go
	./types/gen/gen corev1.Event > types/event/generated_test.go
	./types/gen/gen corev1.Node > types/node/generated_test.go
	./types/gen/gen corev1.ReplicationController > types/replicationcontroller/generated_test.go
	./types/gen/gen appsv1.ReplicaSet > types/replicaset/generated_test.go
	./types/gen/gen appsv1.Deployment > types/deployment/generated_test.go
	./types/gen/gen batchv1.Job > types/job/generated_test.go
	./types/gen/gen appsv1.DaemonSet > types/daemonset/generated_test.go
	./types/gen/gen appsv1.StatefulSet > types/statefulset/generated_test.go
	$(GO) test ./types/...

generate-joins:
	go build -o ./join/gen/gen ./join/gen
	./join/gen/gen Service service '*corev1.Service' Pod pod > ./join/generated_service_pod.go
	./join/gen/gen RC  replicationcontroller '*corev1.ReplicationController' Pod pod > ./join/generated_rc_pod.go
	./join/gen/gen RS  replicaset '*appsv1.ReplicaSet' Pod pod > ./join/generated_rs_pod.go
	./join/gen/gen Deployment deployment '*appsv1.Deployment' Pod pod > ./join/generated_deployment_pod.go
	./join/gen/gen Job job '*batchv1.Job' Pod pod > ./join/generated_job_pod.go
	./join/gen/gen DaemonSet daemonset '*appsv1.DaemonSet' Pod pod > ./join/generated_daemonset_pod.go
	./join/gen/gen Ingress ingress '*networkingv1beta1.Ingress' Service service > ./join/generated_ingress_service.go
	./join/gen/gen StatefulSet statefulset '*appsv1.StatefulSet' Pod pod > ./join/generated_statefulset_pod.go
	$(GO) build ./join

example:
	$(GO) build -o _example/example ./_example

clean:
	rm join/gen/gen types/gen/gen _example/example 2>/dev/null || true

.PHONY: build test test-full install-libs \
	generate generate-types generate-type-tests generate-joins \
	example clean
