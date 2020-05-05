clusters:
- nachtmaar-test-datacontenttype (natss)
- https://dashboard.garden.canary.k8s.ondemand.com/namespace/garden-kyma-dev/shoots/abd6988 (SKR)

# Event-Mesh Health Check Proposal 

Links:
<https://docs.google.com/document/d/1qYnmkRduWLUFQ3vEsaw7jU_mxS_nDHHkDkcGRf1_Fy4>
<https://drive.google.com/file/d/1cJHu68W2R7S6p6h1NEizpHakjonnRTe1/view>

TODO:
- what is health of the event-mesh ??
- do we only wanna now that it is healthy or also what is broken ?
- control-plane vs data-plane vs user-data-plane
  - deadLetterSink ?
 
 
## Overview of Event-Mesh

An overview of the Event-Mesh is illustrated in the following [diagram](https://drive.google.com/open?id=1mCpuz-XQ-aqxT1rYLAEMrhliBK0Ow6uh).
It depicts the control-plane as well as the data-plane.

So what can go wrong in the Event-Mesh ? There might be errors in the control-plane as well as in the data-plane.
In both cases, the user might not be able to send and retrieve events in a subscriber.

Control-Plane errors:

- Eventing related CRs are not created / not healthy
- Eventing related CRs are not healthy
  - e.g errors during reconciliation
  - controller not running
  - controller in error state
- Knative Webhook is not running or returns errors, Knative CRs cannot be created
- Azure EventHubs quota reached

Data Plane errors:
- no events arriving at subscriber
  - service mesh/network might be broken between any two data-plane components => report deadLetterSink event count per application in dashboards
  - kafka consumer/producer not working because of connection problems with Azure EventHubs



1.
=> healthyness: Application.status.installationStatus.status == "DEPLOYED"
2.
=>  same - ApplicationMapping doesn't provide a status - can something go wrong here ??
3.
=> healthyness: ... + foreach ServiceInstance.status.provisionStatus == "Provisioned"
4.
=> action: create trigger and start hearbeats for each enabled namespace
=> healthyness: ... + heartbeat status for application and enabled namespace


## Event-Mesh States

1. Application created
1. Application created | bound to namespace
1. Application created | bound to namespace | Event Class enabled
1. Application created | bound to namespace | Event Class enabled | Trigger created
         ^                         ^                  ^                ^
CRs:  Application      | ApplicationMapping |   ServiceInstance   | Trigger 

## Current health information

Currently we have the following information to determine the health status of the Event-Mesh:

**Dashboards**:
- Kyma / Event Mesh / Broker-Trigger 
- Kyma / Event Mesh / Delivery 
- Kyma / Event Mesh / Latency 

**Alerts**:
- KnEventingSystemPodNotRunning
```promql
sum
  by(pod, namespace) (kube_pod_container_status_running{namespace="knative-eventing",pod!~"(test.*)|((dummy|sample)-.*)|(.*(docs|backup|test)-.*)|((oct-tp-testsuite-all)-.*)|(.*-(tests|dummy))"}
  == 0) * on(pod, namespace) (kube_pod_status_phase{phase="Succeeded"} !=
  1)
```

**k8s objects**:

**CRDs**:
- HTTP Source
- ServiceInstance
- Trigger
- Subscription
- Function

**Data Plane**:
- http source adapter
- channel producer/consumer
- broker-ingress / default-broker

**Control Plane**:
- event-service / compatibility layer
- event-source-controller-manager
  - http source
- knative-eventing-kafka-channel-controller
- knative-eventing-webhook
- application-channel
- Trigger
- knative-eventing-controller
- application-broker
- application-operator

broker-filter / default-broker

## What can go wrong

### Control Plane

- one of the involved CRs not created
- one of the involved CRs not ready
    - e.g HTTPSource not ready
    - e.g ServiceInstance not provisioned successfully

### Data Plane
- Lambdas/microservices don't get triggered by events

## Scenario 1


# Possible Solutions

**Solution 1**: Per application we could check that all required CRs are created and healthy
  - check until Application Channel
    - check HTTPSource healthy
  - check until Broker if ServiceInstance exists  
  - if ServiceInstance exists but is in failed state => unhealthy!  
  - check until subscriber if Trigger is created

**Solution 2**: tracing through data-plane
- based on trace we can tell where event is stuck
- cli tool using same code would also help us debugging event-delivery 
- downside: needs better sampling rate

**Solution 3**: run smoke test (e.g. core-test-external-solution)
- downside: 
  - does not check health status of existing application(s)
  - creates new resources (especially new application) in cluster
- advantage:
  - tests e2e event-flow
  
**Solution 4**: Heartbeat

as soon as:
  - application is created
  - bound to namespace
  - broker is deployed in namespace
  - create a Trigger with subscriber health endpoint
  - send periodic heartbeat events
  - receive them per application
Advantage: tests real event-delivery
Disadvantage: 

## Solution 1

Implement new REST endpoint providing health status for each application 

## Solution 2

Attach health status to application.
Event-Mesh is healthy, if eventing related health status of all applications is healthy.

### Comparison Table

Scenario   | Solution 1 | Solution 2
-------------------------------------
Scenario 1 |      n      |     y
Scenario 2 |      n      |     y
