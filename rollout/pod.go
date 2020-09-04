package rollout

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
)

const WITH_LABEL = false
const WITHOUT_LABEL = true

type patchStringValue struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

// splitLabel - Returns a label and its value
// TODO: Currently only supports one pair. In the future it will support multiple custom labels
func splitLabel(labelList map[string]string) (string, string, error) {
	labelError := errors.New("Empty labels parameter")
	for labelKey, labelVal := range labelList {
		labelError = nil
		return labelKey, labelVal, nil
	}
	return "", "", labelError

}

func (c *Controller) updatePodsLabel(roCtx *canaryContext, pods *corev1.PodList, tempLabels map[string]string) error {
	logCtx := roCtx.Log()
	var updateErr error

	labelKey, labelVal, labelErr := splitLabel(tempLabels)
	if labelErr != nil {
		return labelErr
	}

	for _, pod := range pods.Items {
		fmt.Fprintf(os.Stdout, "pod name: %v\n", pod.Name)

		payload := []patchStringValue{{
			Op:    "replace",
			Path:  "/metadata/labels/" + labelKey,
			Value: labelVal,
		}}
		payloadBytes, _ := json.Marshal(payload)

		_, updateErr = c.kubeclientset.CoreV1().Pods(pod.GetNamespace()).Patch(pod.GetName(), types.JSONPatchType, payloadBytes)
		if updateErr == nil {
			logCtx.Info(fmt.Sprintf("Pod %s label updated successfully.", pod.GetName()))
		} else {
			logCtx.Info(updateErr)
		}
	}

	return updateErr
}

func (c *Controller) removePodsLabel(roCtx *canaryContext, pods *corev1.PodList, tempLabels map[string]string) error {
	logCtx := roCtx.Log()
	var updateErr error

	labelKey, labelVal, labelErr := splitLabel(tempLabels)
	if labelErr != nil {
		return labelErr
	}

	for _, pod := range pods.Items {
		fmt.Fprintf(os.Stdout, "pod name: %v\n", pod.Name)

		payload := []patchStringValue{{
			Op:    "remove",
			Path:  "/metadata/labels/" + labelKey,
			Value: labelVal,
		}}
		payloadBytes, _ := json.Marshal(payload)
		logCtx.Info(string(payloadBytes))

		_, updateErr = c.kubeclientset.CoreV1().Pods(pod.GetNamespace()).Patch(pod.GetName(), types.JSONPatchType, payloadBytes)
		if updateErr == nil {
			logCtx.Infof("Pod %s label removed successfully.", pod.GetName())
		} else {
			logCtx.Warnf("Error remove pod label : %v", updateErr)
		}
	}

	return updateErr
}

func (c *Controller) getPodsForRS(roCtx *canaryContext, rs *appsv1.ReplicaSet, tempLabels map[string]string, negateQuery bool) (*corev1.PodList, error) {

	r := roCtx.Rollout()
	logCtx := roCtx.Log()
	newStatus := c.calculateBaseStatus(roCtx)
	newStatus.Selector = metav1.FormatLabelSelector(rs.Spec.Selector)

	labelSelector := metav1.LabelSelector{MatchLabels: rs.Spec.Selector.MatchLabels}

	labelKey, _, labelErr := splitLabel(tempLabels)
	if labelErr != nil {
		return &corev1.PodList{}, labelErr
	}

	listOptions := metav1.ListOptions{
		Limit: 500,
	}
	if negateQuery == true {
		listOptions.LabelSelector = fmt.Sprintf("!%v,%v", labelKey, labels.Set(labelSelector.MatchLabels).String())
	} else {

		listOptions.LabelSelector = fmt.Sprintf("%v,%v", labelKey, labels.Set(labelSelector.MatchLabels).String())
	}
	logCtx.Infof("list-label-selector [%v]", listOptions.LabelSelector)
	pods, err := c.kubeclientset.CoreV1().Pods(r.Namespace).List(listOptions)

	for _, pod := range pods.Items {
		logCtx.Info(fmt.Sprintf("pod name: %v\n", pod.Name))
	}
	return pods, err
}

// Select canary pods function.

// Select canary pods after sync is complete.
