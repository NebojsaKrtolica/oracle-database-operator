/*
** Copyright (c) 2021 Oracle and/or its affiliates.
**
** The Universal Permissive License (UPL), Version 1.0
**
** Subject to the condition set forth below, permission is hereby granted to any
** person obtaining a copy of this software, associated documentation and/or data
** (collectively the "Software"), free of charge and under any and all copyright
** rights in the Software, and any and all patent rights owned or freely
** licensable by each licensor hereunder covering either (i) the unmodified
** Software as contributed to or provided by such licensor, or (ii) the Larger
** Works (as defined below), to deal in both
**
** (a) the Software, and
** (b) any piece of software and/or hardware listed in the lrgrwrks.txt file if
** one is included with the Software (each a "Larger Work" to which the Software
** is contributed by such licensors),
**
** without restriction, including without limitation the rights to copy, create
** derivative works of, display, perform, and distribute the Software and make,
** use, sell, offer for sale, import, export, have made, and have sold the
** Software and the Larger Work(s), and to sublicense the foregoing rights on
** either these or other terms.
**
** This license is subject to the following condition:
** The above copyright notice and either this complete permission notice or at
** a minimum a reference to the UPL must be included in all copies or
** substantial portions of the Software.
**
** THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
** IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
** FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
** AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
** LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
** OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
** SOFTWARE.
 */

package oci

import (
	"context"
	"math"
	"time"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/oracle/oci-go-sdk/v63/common"
	"github.com/oracle/oci-go-sdk/v63/workrequests"
)

type WorkRequestService interface {
	Get(opcWorkRequestID string) (workrequests.GetWorkRequestResponse, error)
	Wait(opcWorkRequestID string) (workrequests.GetWorkRequestResponse, error)
}

type workRequestService struct {
	logger     logr.Logger
	workClient workrequests.WorkRequestClient
}

func NewWorkRequestService(
	logger logr.Logger,
	kubeClient client.Client,
	provider common.ConfigurationProvider) (WorkRequestService, error) {

	workClient, err := workrequests.NewWorkRequestClientWithConfigurationProvider(provider)
	if err != nil {
		return nil, err
	}

	return &workRequestService{
		logger:     logger,
		workClient: workClient,
	}, nil
}

func (w workRequestService) getRetryPolicy() common.RetryPolicy {
	shouldRetry := func(r common.OCIOperationResponse) bool {
		if _, isServiceError := common.IsServiceError(r.Error); isServiceError {
			// Don't retry if it's service error. Sometimes it could be network error or other errors which prevents
			// request send to server; we do the retry in these cases.
			return false
		}

		if converted, ok := r.Response.(workrequests.GetWorkRequestResponse); ok {
			// do the retry until WorkReqeut Status is Succeeded  - ignore case (BMI-2652)
			return converted.Status != workrequests.WorkRequestStatusSucceeded &&
				converted.Status != workrequests.WorkRequestStatusFailed &&
				converted.Status != workrequests.WorkRequestStatusCanceled
		}

		return true
	}

	// maximum times of retry (~60mins)
	attempts := uint(124)

	nextDuration := func(r common.OCIOperationResponse) time.Duration {
		// Wait longer for next retry when your previous one failed
		// this function will return the duration as:
		// 1s, 2s, 4s, 8s, 16s, 30s, 30s etc...
		if r.AttemptNumber <= 5 {
			return time.Duration(math.Pow(float64(2), float64(r.AttemptNumber-1))) * time.Second
		}
		return time.Duration(30) * time.Second
	}

	return common.NewRetryPolicy(attempts, shouldRetry, nextDuration)
}

func (w *workRequestService) Wait(opcWorkRequestID string) (workrequests.GetWorkRequestResponse, error) {
	w.logger.Info("Waiting for the work request to finish. opcWorkRequestID = " + opcWorkRequestID)

	// retries until the work status is SUCCEEDED, FAILED or CANCELED
	retryPolicy := w.getRetryPolicy()

	workRequest := workrequests.GetWorkRequestRequest{
		WorkRequestId: common.String(opcWorkRequestID),
		RequestMetadata: common.RequestMetadata{
			RetryPolicy: &retryPolicy,
		},
	}

	resp, err := w.workClient.GetWorkRequest(context.TODO(), workRequest)
	if err != nil {
		return resp, err
	}

	return resp, nil
}

func (w *workRequestService) Get(opcWorkRequestID string) (workrequests.GetWorkRequestResponse, error) {
	workRequest := workrequests.GetWorkRequestRequest{
		WorkRequestId: common.String(opcWorkRequestID),
	}

	resp, err := w.workClient.GetWorkRequest(context.TODO(), workRequest)
	if err != nil {
		return resp, err
	}

	return resp, nil
}