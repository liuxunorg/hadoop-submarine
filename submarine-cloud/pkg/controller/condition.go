package controller

import (
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	rapi "github.com/apache/submarine/submarine-cloud/pkg/apis/submarine/v1alpha1"
)

func setRebalancingCondition(clusterStatus *rapi.SubmarineClusterStatus, status bool) bool {
	statusCondition := apiv1.ConditionFalse
	if status {
		statusCondition = apiv1.ConditionTrue
	}
	return setCondition(clusterStatus, rapi.SubmarineClusterRebalancing, statusCondition, metav1.Now(), "topology as changed", "reconfigure on-going after topology changed")
}

func setCondition(clusterStatus *rapi.SubmarineClusterStatus, conditionType rapi.SubmarineClusterConditionType, status apiv1.ConditionStatus, now metav1.Time, reason, message string) bool {
	updated := false
	found := false
	for i, c := range clusterStatus.Conditions {
		if c.Type == conditionType {
			found = true
			if c.Status != status {
				updated = true
				clusterStatus.Conditions[i] = updateCondition(c, status, now, reason, message)
			}
		}
	}
	if !found {
		updated = true
		clusterStatus.Conditions = append(clusterStatus.Conditions, newCondition(conditionType, status, now, reason, message))
	}
	return updated
}

func setRollingUpdategCondition(clusterStatus *rapi.SubmarineClusterStatus, status bool) bool {
	statusCondition := apiv1.ConditionFalse
	if status {
		statusCondition = apiv1.ConditionTrue
	}
	return setCondition(clusterStatus, rapi.SubmarineClusterRollingUpdate, statusCondition, metav1.Now(), "Rolling update ongoing", "a Rolling update is ongoing")
}

func setScalingCondition(clusterStatus *rapi.SubmarineClusterStatus, status bool) bool {
	statusCondition := apiv1.ConditionFalse
	if status {
		statusCondition = apiv1.ConditionTrue
	}
	return setCondition(clusterStatus, rapi.SubmarineClusterScaling, statusCondition, metav1.Now(), "cluster needs more pods", "cluster needs more pods")
}

// updateCondition return an updated version of the RedisClusterCondition
func updateCondition(from rapi.SubmarineClusterCondition, status apiv1.ConditionStatus, now metav1.Time, reason, message string) rapi.SubmarineClusterCondition {
	newCondition := from.DeepCopy()
	newCondition.LastProbeTime = now
	newCondition.Message = message
	newCondition.Reason = reason
	if status != newCondition.Status {
		newCondition.Status = status
		newCondition.LastTransitionTime = now
	}

	return *newCondition
}

// newCondition return a new defaulted instance of a RedisClusterCondition
func newCondition(conditionType rapi.SubmarineClusterConditionType, status apiv1.ConditionStatus, now metav1.Time, reason, message string) rapi.SubmarineClusterCondition {
	return rapi.SubmarineClusterCondition{
		Type:               conditionType,
		Status:             status,
		LastProbeTime:      now,
		LastTransitionTime: now,
		Reason:             reason,
		Message:            message,
	}
}

func setClusterStatusCondition(clusterStatus *rapi.SubmarineClusterStatus, status bool) bool {
	statusCondition := apiv1.ConditionFalse
	if status {
		statusCondition = apiv1.ConditionTrue
	}
	return setCondition(clusterStatus, rapi.SubmarineClusterOK, statusCondition, metav1.Now(), "redis-cluster is correctly configure", "redis-cluster is correctly configure")
}
