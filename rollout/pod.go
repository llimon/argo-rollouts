package rollout

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
)

func getPodControlID(pod *corev1.Pod) string {

	if pod.Labels == nil {
		return string(pod.UID)
	} else {
		uuid, ok := pod.Labels["controller-uid"]
		if ok {
			return uuid
		} else {
			return string(pod.UID)
		}
	}
}

type patchStringValue struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

func (c *Controller) updatePodsLabel(roCtx *canaryContext, pods *corev1.PodList) error {
	logCtx := roCtx.Log()
	var updateErr error

	for _, pod := range pods.Items {
		fmt.Fprintf(os.Stdout, "pod name: %v\n", pod.Name)

		payload := []patchStringValue{{
			Op:    "replace",
			Path:  "/metadata/labels/rollout_stage",
			Value: time.Now().Format("canary"),
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

func (c *Controller) removePodsLabel(roCtx *canaryContext, pods *corev1.PodList) error {
	logCtx := roCtx.Log()
	var updateErr error

	for _, pod := range pods.Items {
		fmt.Fprintf(os.Stdout, "pod name: %v\n", pod.Name)

		payload := []patchStringValue{{
			Op:    "remove",
			Path:  "/metadata/labels/rollout_stage",
			Value: time.Now().Format("canary"),
		}}
		payloadBytes, _ := json.Marshal(payload)

		_, updateErr = c.kubeclientset.CoreV1().Pods(pod.GetNamespace()).Patch(pod.GetName(), types.JSONPatchType, payloadBytes)
		if updateErr == nil {
			logCtx.Info(fmt.Sprintf("Pod %s label removed successfully.", pod.GetName()))
		} else {
			logCtx.Info(updateErr)
		}
	}

	return updateErr
}

func (c *Controller) getPodsForRS(roCtx *canaryContext) (*corev1.PodList, error) {

	r := roCtx.Rollout()
	logCtx := roCtx.Log()
	//newRS := roCtx.NewRS()
	//stableRS := roCtx.StableRS()
	newStatus := c.calculateBaseStatus(roCtx)
	newStatus.Selector = metav1.FormatLabelSelector(r.Spec.Selector)

	labelSelector := metav1.LabelSelector{MatchLabels: r.Spec.Selector.MatchLabels}

	//labelSelector.MatchLabels["!luis"] = "canary"

	fmt.Println(labelSelector)

	// TODO - Investigate if a better way to add multiple selectors with negation.
	listOptions := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("!rollout_stage,%v", labels.Set(labelSelector.MatchLabels).String()),
		Limit:         100,
	}

	pods, err := c.kubeclientset.CoreV1().Pods(r.Namespace).List(listOptions)

	for _, pod := range pods.Items {
		logCtx.Info(fmt.Sprintf("pod name: %v\n", pod.Name))
	}
	return pods, err
}

// Select canary pods function.

// Select canary pods after sync is complete.
