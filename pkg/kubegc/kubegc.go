package kubegc

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	selection "k8s.io/apimachinery/pkg/selection"
	sets "k8s.io/apimachinery/pkg/util/sets"
	discovery "k8s.io/client-go/discovery"
	dynamic "k8s.io/client-go/dynamic"
	rest "k8s.io/client-go/rest"
)

// KubeGC interface
type KubeGC interface {
    Clean(context.Context, bool) error
}

type kubeGC struct {
    config          *rest.Config
    namespace       string
    labelSelector   string
    annotationFilterName  string
    annotationFilterValue string
}

type orphanResource struct {
    gv        schema.GroupVersion
    kind      string
    resource  string
    namespace string
    name      string
}

// NewKubeGC creates new kubeGC instance
func NewKubeGC(config *rest.Config, namespace string, labelSelector string, annotationFilter string) (KubeGC, error) {
    annotationFilterName := ""
    annotationFilterValue := ""
    if annotationFilter != "" {
        filter := strings.Split(annotationFilter, "=")
        if len(filter) != 2 {
            return nil, fmt.Errorf("'annotationFilter' string cannot be parsed (has to be 'name=value')")
        }
        annotationFilterName = strings.TrimSpace(filter[0])
        annotationFilterValue = strings.TrimSpace(filter[1])
    }

    /*
    ensure label selector key exists, so that selector like
    'test=test' will return only resources with label 'test' set
    */

    selector, err := labels.Parse(labelSelector)
    if err != nil {
        return nil, err
    }

    requirements, _ := selector.Requirements()
    for _, req := range requirements {
        addReq, err := labels.NewRequirement(req.Key(), selection.Exists, []string{})
        if err != nil {
            return nil, err
        }
        selector = selector.Add(*addReq)
    }

    return &kubeGC{
        config:           config,
        namespace:        namespace,
        labelSelector:    selector.String(),
        annotationFilterName: annotationFilterName,
        annotationFilterValue: annotationFilterValue,
        }, nil
}

func (gc *kubeGC) String() string {
    scope := gc.namespace
    if scope == "" {
        scope = "Cluster wide"
    }

    return fmt.Sprintf(
        "Scope: '%s', LabelSelector: '%s', AnnotationFilter: '%s=%s'",
        scope,
        gc.labelSelector,
        gc.annotationFilterName, gc.annotationFilterValue,
        )
}

func (gc *kubeGC) Clean(context context.Context, dryRun bool) error {
    dynamicClient, err := dynamic.NewForConfig(gc.config)
    if err != nil {
        return err
    }

    discoveryClient, err := discovery.NewDiscoveryClientForConfig(gc.config)
    if err != nil {
        return err
    }

    serverGroups, err := discoveryClient.ServerGroups()
    if err != nil {
        return err
    }

    var orphans = []*orphanResource{}

    for _, group := range serverGroups.Groups {
        srs, err := discoveryClient.ServerResourcesForGroupVersion(group.PreferredVersion.GroupVersion)
        if err != nil {
            log.Print(err)
            continue
        }
        gv, _ := schema.ParseGroupVersion(group.PreferredVersion.GroupVersion)
        for _, apiResource := range srs.APIResources {
            if !sets.NewString([]string(apiResource.Verbs)...).HasAll("delete") {
                continue
            }

            list, err := dynamicClient.
                Resource(
                    schema.GroupVersionResource{
                        Group:    gv.Group,
                        Version:  gv.Version,
                        Resource: apiResource.Name,
                        },
                        ).
                Namespace(gc.namespace).
                List(
                    context,
                    metav1.ListOptions{
                        LabelSelector: gc.labelSelector,
                        },
                        )

            if err != nil {
                //log.Print(err)
                continue
            }
            for _, r := range list.Items {
                if len(r.GetOwnerReferences()) > 0 {
                    continue
                }

                if gc.annotationFilterName != "" {
                    annotations := r.GetAnnotations()
                    value, exist := annotations[gc.annotationFilterName]
                    if ! exist || value != gc.annotationFilterValue {
                        continue
                    }
                }

                orphans = append(orphans, &orphanResource{
                    gv:        gv,
                    resource:  apiResource.Name,
                    kind:      apiResource.Kind,
                    namespace: r.GetNamespace(),
                    name:      r.GetName(),
                    })
            }
        }
    }

    // Sort resources list: non-namespaced, namespaced, namespaces, CRDs
    weight := func(o *orphanResource) int {
        if o.kind == "CustomResourceDefinition" {
            return 4
        }
        if o.kind == "Namespace" {
            return 3
        }
        if o.namespace != "" {
            return 2
        }
        return 1
    }
    sort.Slice(orphans, func(i, j int) bool {
        return weight(orphans[i]) < weight(orphans[j])
    })

    // Do cleanup
    logPrefix := ""
    deletePolicy := metav1.DeletePropagationForeground
    deleteOptions := metav1.DeleteOptions{
        PropagationPolicy: &deletePolicy,
        }
        if dryRun {
            deleteOptions.DryRun = []string{metav1.DryRunAll}
            logPrefix = "(dry-run) "
        }

        for _, o := range orphans {
            result := "OK"
            err = dynamicClient.
                Resource(o.gv.WithResource(o.resource)).
                Namespace(o.namespace).
                Delete(context, o.name, deleteOptions)
            if err != nil {
                result = err.Error()
            }

            if o.namespace == "" {
                log.Printf("%sdelete %s/%s... %s", logPrefix, o.resource, o.name, result)
            } else {
                log.Printf("%sdelete %s/%s in namespace %s... %s", logPrefix, o.resource, o.name, o.namespace, result)
            }
        }

        return nil
}
