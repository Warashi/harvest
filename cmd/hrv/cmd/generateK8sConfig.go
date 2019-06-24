/*
Copyright © 2019 Ken'ichiro Oyama <k1lowxb@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/k1LoW/harvest/client/k8s"
	"github.com/k1LoW/harvest/config"
	"github.com/k1LoW/harvest/logger"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	contextName string
	namespace   string
)

// generateK8sConfigCmd represents the generateK8sConfig command
var generateK8sConfigCmd = &cobra.Command{
	Use:   "generate-k8s-config",
	Short: "generate harvest config.yml via Kubernetes cluster",
	Long:  `generate harvest config.yml via Kubernetes cluster.`,
	Run: func(cmd *cobra.Command, args []string) {
		l := logger.NewLogger()

		if contextName == "" {
			cc, err := k8s.GetCurrentContext()
			if err != nil {
				l.Error("kube config error", zap.String("error", err.Error()))
				os.Exit(1)
			}
			contextName = cc
		}

		clientset, err := k8s.NewKubeClientSet(contextName)
		if err != nil {
			l.Error("kube config error", zap.String("error", err.Error()))
			os.Exit(1)
		}

		list, err := clientset.CoreV1().Pods(namespace).List(metav1.ListOptions{})
		if err != nil {
			l.Error("error", zap.String("error", err.Error()))
			os.Exit(1)
		}

		c, err := config.NewConfig()
		if err != nil {
			l.Error("error", zap.String("error", err.Error()))
			os.Exit(1)
		}

		re := regexp.MustCompile(`[+\-*\/%.]`)
		reNumber := regexp.MustCompile(`^\d+$`)
		tagetSetMap := map[string]struct{}{}

		for _, i := range list.Items {
			source := strings.Join([]string{"k8s:/", contextName, i.ObjectMeta.Namespace, fmt.Sprintf("%s*", i.ObjectMeta.GenerateName)}, "/")
			if _, ok := tagetSetMap[source]; ok {
				continue
			} else {
				tagetSetMap[source] = struct{}{}
			}
			tags := []string{re.ReplaceAllString(contextName, "_"), re.ReplaceAllString(i.ObjectMeta.Namespace, "_")}
			for _, v := range i.ObjectMeta.Labels {
				if reNumber.MatchString(v) {
					continue
				}
				switch v {
				case "true", "false":
					continue
				default:
					tags = append(tags, re.ReplaceAllString(v, "_"))
				}
			}
			c.TargetSets = append(c.TargetSets, &config.TargetSet{
				Sources:     []string{source},
				Description: "Generated by `hrv generate-k8s-config`",
				Type:        "k8s",
				MultiLine:   false,
				Tags:        uniqueTags(tags),
			})
		}
		y, err := yaml.Marshal(&c)
		if err != nil {
			l.Error("generate error", zap.String("error", err.Error()))
			os.Exit(1)
		}
		fmt.Printf("%s\n", string(y))
	},
}

func init() {
	rootCmd.AddCommand(generateK8sConfigCmd)
	generateK8sConfigCmd.Flags().StringVarP(&contextName, "context", "c", "", "kubernetes context. default:current context")
	generateK8sConfigCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "kubernetes namespace")
}

func uniqueTags(tags []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, t := range tags {
		if _, value := keys[t]; !value {
			keys[t] = true
			list = append(list, t)
		}
	}
	return list
}
