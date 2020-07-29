// Copyright 2017 The Kubernetes Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package parser

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	metricapi "github.com/ycyxuehan/dashboard-gin/backend/integration/metric/api"
	"github.com/ycyxuehan/dashboard-gin/backend/resource/dataselect"
)

func parsePaginationPathParameter(c *gin.Context) *dataselect.PaginationQuery {
	itemsPerPage, err := strconv.ParseInt(c.Query("itemsPerPage"), 10, 0)
	if err != nil {
		return dataselect.NoPagination
	}

	page, err := strconv.ParseInt(c.Query("page"), 10, 0)
	if err != nil {
		return dataselect.NoPagination
	}

	// Frontend pages start from 1 and backend starts from 0
	return dataselect.NewPaginationQuery(int(itemsPerPage), int(page-1))
}

func parseFilterPathParameter(c *gin.Context) *dataselect.FilterQuery {
	return dataselect.NewFilterQuery(strings.Split(c.Query("filterBy"), ","))
}

// Parses query parameters of the request and returns a SortQuery object
func parseSortPathParameter(c *gin.Context) *dataselect.SortQuery {
	return dataselect.NewSortQuery(strings.Split(c.Query("sortBy"), ","))
}

// Parses query parameters of the request and returns a MetricQuery object
func parseMetricPathParameter(c *gin.Context) *dataselect.MetricQuery {
	metricNamesParam := c.Query("metricNames")
	var metricNames []string
	if metricNamesParam != "" {
		metricNames = strings.Split(metricNamesParam, ",")
	} else {
		metricNames = nil
	}
	aggregationsParam := c.Query("aggregations")
	var rawAggregations []string
	if aggregationsParam != "" {
		rawAggregations = strings.Split(aggregationsParam, ",")
	} else {
		rawAggregations = nil
	}
	aggregationModes := metricapi.AggregationModes{}
	for _, e := range rawAggregations {
		aggregationModes = append(aggregationModes, metricapi.AggregationMode(e))
	}
	return dataselect.NewMetricQuery(metricNames, aggregationModes)

}

// ParseDataSelectPathParameter parses query parameters of the request and returns a DataSelectQuery object
func ParseDataSelectPathParameter(c *gin.Context) *dataselect.DataSelectQuery {
	paginationQuery := parsePaginationPathParameter(c)
	sortQuery := parseSortPathParameter(c)
	filterQuery := parseFilterPathParameter(c)
	metricQuery := parseMetricPathParameter(c)
	return dataselect.NewDataSelectQuery(paginationQuery, sortQuery, filterQuery, metricQuery)
}
