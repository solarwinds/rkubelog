/*
 * AUTO GENERATED - DO NOT EDIT BY HAND
 */

package join

import (
	"context"

	logutil "github.com/boz/go-logutil"
	"github.com/boz/kcache/filter"
	"github.com/boz/kcache/types/job"
	"github.com/boz/kcache/types/pod"
	batchv1 "k8s.io/api/batch/v1"
)

func JobPodsWith(ctx context.Context,
	srcController job.Controller,
	dstController pod.Publisher,
	filterFn func(...*batchv1.Job) filter.ComparableFilter) (pod.Controller, error) {

	log := logutil.FromContextOrDefault(ctx)

	dst, err := dstController.CloneForFilter()
	if err != nil {
		return nil, err
	}

	update := func(_ *batchv1.Job) {
		objs, err := srcController.Cache().List()
		if err != nil {
			log.Err(err, "join(job,pod: cache list")
			return
		}
		dst.Refilter(filterFn(objs...))
	}

	handler := job.BuildHandler().
		OnInitialize(func(objs []*batchv1.Job) { dst.Refilter(filterFn(objs...)) }).
		OnCreate(update).
		OnUpdate(update).
		OnDelete(update).
		Create()

	monitor, err := job.NewMonitor(srcController, handler)
	if err != nil {
		dst.Close()
		return nil, log.Err(err, "join(job,pod): monitor")
	}

	go func() {
		<-dst.Done()
		monitor.Close()
	}()

	return dst, nil
}
