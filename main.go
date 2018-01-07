package main

import (
	"fmt"

	"time"

	"flag"
	"path/filepath"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// PoC to create a new k8s Job, making use of Init Containers
// Ref
//   https://kubernetes.io/docs/concepts/workloads/pods/init-containers/
func main() {
	var kubeConfig *string
	var kubeNamespace = "poc-kubernetes-batch"

	var resultsMountPath = "/tmp/results"
	var resultsFile = resultsMountPath + "/fromInitContainer"

	if home := homedir.HomeDir(); home != "" {
		kubeConfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeConfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}

	// Ref
	//   https://godoc.org/k8s.io/client-go/tools/clientcmd#BuildConfigFromFlags
	//   https://godoc.org/k8s.io/client-go/rest#Config
	config, err := clientcmd.BuildConfigFromFlags("", *kubeConfig)
	if err != nil {
		panic(err.Error())
	}

	// Ref
	//   https://godoc.org/k8s.io/client-go/kubernetes#NewForConfig
	//   https://godoc.org/k8s.io/client-go/kubernetes#Clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// Define our job
	// Ref
	//   https://godoc.org/k8s.io/api/core/v1
	//   https://godoc.org/k8s.io/api/batch/v1
	//   https://godoc.org/k8s.io/apimachinery/pkg/apis/meta/v1

	var envVars = []corev1.EnvVar{
		{
			Name:  "FOO",
			Value: "FOOVAL",
		},
		{
			Name:  "BAR",
			Value: "BARVAL",
		},
	}

	var jobDefinition = batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "poc-job",
		},
		Spec: batchv1.JobSpec{
			Parallelism: int32Ptr(1),
			Completions: int32Ptr(1),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":       "jobWorker",
						"component": "jobs",
					},
					Namespace: kubeNamespace,
				},
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name:            "poc-init-container",
							Image:           "alpine:latest",
							ImagePullPolicy: "IfNotPresent",
							Args: []string{
								"sh", "-c",
								fmt.Sprintf("echo Hello Init Container World! > %s && cat %s", resultsFile, resultsFile)},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "results",
									MountPath: resultsMountPath,
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:            "poc-main-container",
							Image:           "alpine:latest",
							ImagePullPolicy: "IfNotPresent",
							Env:             envVars,
							Args: []string{
								"sh", "-c",
								fmt.Sprintf("echo Hello Main Container World! && cat %s", resultsFile)},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "results",
									MountPath: resultsMountPath,
								},
							},
						},
					},
					RestartPolicy: "OnFailure",
					Volumes: []corev1.Volume{
						{
							Name: "results",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									Medium: "",
								},
							},
						},
					},
				},
			},
		},
	}

	// Create the job
	// Ref
	//   https://godoc.org/k8s.io/client-go/kubernetes#Clientset.BatchV1
	//   https://godoc.org/k8s.io/client-go/kubernetes/typed/batch/v1#JobsGetter
	//   https://godoc.org/k8s.io/client-go/kubernetes/typed/batch/v1#JobInterface
	job, err := clientset.BatchV1().Jobs(kubeNamespace).Create(&jobDefinition)
	if err != nil {
		panic(err.Error())
	}

	fmt.Printf("Created Job: %s\n", job.Name)

	// Wait for the job to complete
	for {
		checkJob, err := clientset.BatchV1().Jobs(kubeNamespace).Get(job.Name, metav1.GetOptions{})
		if err != nil {
			panic(err.Error())
		}

		jobStatus := checkJob.Status.Succeeded
		fmt.Printf("Job: %s, Status: %d\n", job.Name, jobStatus)

		if jobStatus == 1 {
			fmt.Println("Job Completed")
			break
		}
		time.Sleep(1 * time.Second)
	}

	// List the pods in the current namespace
	// Ref
	//   https://godoc.org/k8s.io/client-go/kubernetes/typed/core/v1#PodInterface
	//   https://godoc.org/k8s.io/apimachinery/pkg/apis/meta/v1#ListOptions
	pods, err := clientset.CoreV1().Pods(kubeNamespace).List(metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	// Get logs from the pods
	// Ref
	//   https://godoc.org/k8s.io/client-go/kubernetes/typed/core/v1#PodExpansion
	//   https://godoc.org/k8s.io/api/core/v1#PodLogOptions
	for _, pod := range pods.Items {
		logsReq := clientset.CoreV1().Pods(kubeNamespace).GetLogs(pod.Name, &corev1.PodLogOptions{})
		logs, err := logsReq.Do().Raw()
		if err != nil {
			panic(err.Error())
		}

		fmt.Printf("Logs for %s:\n=====\n%s\n=====\n", pod.Name, logs)
	}

	// Once the job has finished, let's delete it (and the pods)

	cleanupJob, err := clientset.BatchV1().Jobs(kubeNamespace).Get(job.Name, metav1.GetOptions{})
	if err != nil {
		panic(err.Error())
	}

	if cleanupJob.Status.Succeeded == 1 {
		fmt.Printf("Deleting Job: %s\n", cleanupJob.Name)
		clientset.BatchV1().Jobs(kubeNamespace).Delete(cleanupJob.Name, &metav1.DeleteOptions{})

		fmt.Println("Deleting completed pods..")
		// TODO: Is there a better way to get only the pods associated with our job?
		for _, pod := range pods.Items {
			// TODO: Can we improve this if statement by only selecting the correct pods with the ListOptions above
			if pod.Status.Phase == "Succeeded" && pod.Labels["app"] == "jobWorker" && pod.Labels["component"] == "jobs" {
				fmt.Printf("Deleting Job Pod: %s, Status: %s\n", pod.Name, pod.Status.Phase)
				clientset.CoreV1().Pods(kubeNamespace).Delete(pod.Name, &metav1.DeleteOptions{})
			}
		}
	}
}

func int32Ptr(i int32) *int32 { return &i }
