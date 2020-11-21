package main

import (
	"context"
	"log"
	"os"
	"path"

	"k8s-utils/pkg/kubegc"

	flag "github.com/spf13/pflag"
	clientcmd "k8s.io/client-go/tools/clientcmd"
)

func main() {
	var kubeConfig string
	var labelSelector string
	var namespace string
	var annotationFilter string
	var dryRun bool

	flag.StringVar(&kubeConfig, "kubeconfig", "", "kubeconfig file")
	flag.StringVar(&namespace, "namespace", "", "limit cleanup to a particular namespace")
	flag.StringVar(&labelSelector, "label-selector", "", "resources to prune (required)")
	flag.StringVar(&annotationFilter, "annotation-filter", "", "preserve annotated resources")
	flag.BoolVar(&dryRun, "dry-run", true, "report only (default 'true')")
	flag.Parse()

	if len(labelSelector) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	if len(kubeConfig) == 0 {
		kubeConfig = os.Getenv("KUBECONFIG")
		if len(kubeConfig) == 0 {
			kubeConfigDefaultPath := path.Join(os.Getenv("HOME"), ".kube/config")
			if _, err := os.Stat(kubeConfigDefaultPath); err == nil {
				kubeConfig = kubeConfigDefaultPath
			}
		}
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}

	config.Burst = 100

	gc, err := kubegc.NewKubeGC(config, namespace, labelSelector, annotationFilter)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}

	log.Printf("Running GC with: %v", gc)

	err = gc.Clean(context.Background(), dryRun)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}

	log.Printf("Done")
}
