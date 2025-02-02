package tests

import (
	"fmt"
	opsapi "github.com/libopenstorage/openstorage/api"
	"github.com/portworx/torpedo/pkg/log"
	"math/rand"
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/portworx/sched-ops/k8s/apps"
	"github.com/portworx/torpedo/drivers/node"
	"github.com/portworx/torpedo/drivers/scheduler"
	"github.com/portworx/torpedo/pkg/testrailuttils"
	. "github.com/portworx/torpedo/tests"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// This test performs basic test of starting an application and destroying it (along with storage)
var _ = Describe("{SetupTeardown}", func() {
	var testrailID = 35258
	// testrailID corresponds to: https://portworx.testrail.net/index.php?/cases/view/35258
	var runID int
	JustBeforeEach(func() {
		StartTorpedoTest("SetupTeardown", "Validate setup tear down", nil, testrailID)
		runID = testrailuttils.AddRunsToMilestone(testrailID)
	})
	var contexts []*scheduler.Context

	It("has to setup, validate and teardown apps", func() {
		contexts = make([]*scheduler.Context, 0)

		for i := 0; i < Inst().GlobalScaleFactor; i++ {
			contexts = append(contexts, ScheduleApplications(fmt.Sprintf("setupteardown-%d", i))...)
		}
		ValidateApplications(contexts)

		opts := make(map[string]bool)
		opts[scheduler.OptionsWaitForResourceLeakCleanup] = true

		for _, ctx := range contexts {
			TearDownContext(ctx, opts)
		}
	})
	JustAfterEach(func() {
		defer EndTorpedoTest()
		AfterEachTest(contexts, testrailID, runID)
	})
})

// Volume Driver Plugin is down, unavailable - and the client container should not be impacted.
var _ = Describe("{VolumeDriverDown}", func() {
	var testrailID = 35259
	// testrailID corresponds to: https://portworx.testrail.net/index.php?/cases/view/35259
	var runID int
	JustBeforeEach(func() {
		StartTorpedoTest("VolumeDriverDown", "Validate volume driver down", nil, testrailID)
		runID = testrailuttils.AddRunsToMilestone(testrailID)
	})
	var contexts []*scheduler.Context

	stepLog := "has to schedule apps and stop volume driver on app nodes"
	It(stepLog, func() {
		log.InfoD(stepLog)
		contexts = make([]*scheduler.Context, 0)

		for i := 0; i < Inst().GlobalScaleFactor; i++ {
			contexts = append(contexts, ScheduleApplications(fmt.Sprintf("voldriverdown-%d", i))...)
		}

		ValidateApplications(contexts)

		Step("get nodes bounce volume driver", func() {
			for _, appNode := range node.GetStorageDriverNodes() {
				stepLog = fmt.Sprintf("stop volume driver %s on node: %s",
					Inst().V.String(), appNode.Name)
				Step(stepLog,
					func() {
						log.InfoD(stepLog)
						StopVolDriverAndWait([]node.Node{appNode})
					})

				stepLog = fmt.Sprintf("starting volume %s driver on node %s",
					Inst().V.String(), appNode.Name)
				Step(stepLog,
					func() {
						log.InfoD(stepLog)
						StartVolDriverAndWait([]node.Node{appNode})
					})

				stepLog = "Giving few seconds for volume driver to stabilize"
				Step(stepLog, func() {
					log.InfoD(stepLog)
					time.Sleep(20 * time.Second)
				})

				Step("validate apps", func() {
					for _, ctx := range contexts {
						ValidateContext(ctx)
					}
				})
			}
		})

		Step("destroy apps", func() {
			opts := make(map[string]bool)
			opts[scheduler.OptionsWaitForResourceLeakCleanup] = true
			for _, ctx := range contexts {
				TearDownContext(ctx, opts)
			}
		})
	})
	JustAfterEach(func() {
		defer EndTorpedoTest()
		AfterEachTest(contexts, testrailID, runID)
	})
})

// Volume Driver Plugin is down, unavailable on the nodes where the volumes are
// attached - and the client container should not be impacted.
var _ = Describe("{VolumeDriverDownAttachedNode}", func() {
	var testrailID = 35260
	// testrailID corresponds to: https://portworx.testrail.net/index.php?/cases/view/35260
	var runID int
	JustBeforeEach(func() {
		StartTorpedoTest("VolumeDriverDownAttachedNode", "Validate Volume drive down on an volume attached node", nil, testrailID)
		runID = testrailuttils.AddRunsToMilestone(testrailID)
	})
	var contexts []*scheduler.Context

	stepLog := "has to schedule apps and stop volume driver on nodes where volumes are attached"
	It(stepLog, func() {
		log.InfoD(stepLog)
		contexts = make([]*scheduler.Context, 0)

		for i := 0; i < Inst().GlobalScaleFactor; i++ {
			contexts = append(contexts, ScheduleApplications(fmt.Sprintf("voldriverdownattachednode-%d", i))...)
		}

		ValidateApplications(contexts)

		stepLog = "get nodes where app is running and restart volume driver"
		Step(stepLog, func() {
			log.InfoD(stepLog)
			for _, ctx := range contexts {
				appNodes, err := Inst().S.GetNodesForApp(ctx)
				dash.VerifySafely(err, nil, fmt.Sprintf("Verify Get nodes for app %s", ctx.App.Key))
				for _, appNode := range appNodes {
					stepLog = fmt.Sprintf("stop volume driver %s on app %s's node: %s",
						Inst().V.String(), ctx.App.Key, appNode.Name)
					Step(stepLog,
						func() {
							StopVolDriverAndWait([]node.Node{appNode})
						})

					stepLog = fmt.Sprintf("starting volume %s driver on app %s's node %s",
						Inst().V.String(), ctx.App.Key, appNode.Name)
					Step(stepLog,
						func() {
							StartVolDriverAndWait([]node.Node{appNode})
						})

					stepLog = "Giving few seconds for volume driver to stabilize"
					Step(stepLog, func() {
						log.InfoD("Giving few seconds for volume driver to stabilize")
						time.Sleep(20 * time.Second)
					})

					stepLog = fmt.Sprintf("validate app %s", ctx.App.Key)
					Step(stepLog, func() {
						log.InfoD(stepLog)
						ValidateContext(ctx)
					})
				}
			}
		})

		Step("destroy apps", func() {
			opts := make(map[string]bool)
			opts[scheduler.OptionsWaitForResourceLeakCleanup] = true
			for _, ctx := range contexts {
				TearDownContext(ctx, opts)
			}
		})
	})
	JustAfterEach(func() {
		defer EndTorpedoTest()
		AfterEachTest(contexts, testrailID, runID)
	})
})

// Volume Driver Plugin has crashed - and the client container should not be impacted.
var _ = Describe("{VolumeDriverCrash}", func() {
	var testrailID = 35261
	// testrailID corresponds to: https://portworx.testrail.net/index.php?/cases/view/35261
	var runID int
	JustBeforeEach(func() {
		StartTorpedoTest("VolumeDriverCrash", "Validate PX after volume driver crash", nil, testrailID)
		runID = testrailuttils.AddRunsToMilestone(testrailID)
	})
	var contexts []*scheduler.Context

	stepLog := "has to schedule apps and crash volume driver on app nodes"
	It(stepLog, func() {
		log.InfoD(stepLog)
		contexts = make([]*scheduler.Context, 0)

		for i := 0; i < Inst().GlobalScaleFactor; i++ {
			contexts = append(contexts, ScheduleApplications(fmt.Sprintf("voldrivercrash-%d", i))...)
		}

		ValidateApplications(contexts)

		stepLog = "crash volume driver in all nodes"
		Step(stepLog, func() {
			log.InfoD(stepLog)
			for _, appNode := range node.GetStorageDriverNodes() {
				stepLog = fmt.Sprintf("crash volume driver %s on node: %v",
					Inst().V.String(), appNode.Name)
				Step(stepLog,
					func() {
						log.InfoD(stepLog)
						CrashVolDriverAndWait([]node.Node{appNode})
					})
			}
		})

		opts := make(map[string]bool)
		opts[scheduler.OptionsWaitForResourceLeakCleanup] = true
		ValidateAndDestroy(contexts, opts)
	})
	JustAfterEach(func() {
		defer EndTorpedoTest()
		AfterEachTest(contexts, testrailID, runID)
	})
})

// Volume driver plugin is down and the client container gets terminated.
// There is a lost unmount call in this case. When the volume driver is
// back up, we should be able to detach and delete the volume.
var _ = Describe("{VolumeDriverAppDown}", func() {
	var testrailID = 35262
	// testrailID corresponds to: https://portworx.testrail.net/index.php?/cases/view/35262
	var runID int
	JustBeforeEach(func() {
		StartTorpedoTest("VolumeDriverAppDown", "Validate volume driver down and app deletion", nil, testrailID)
		runID = testrailuttils.AddRunsToMilestone(testrailID)
	})
	var contexts []*scheduler.Context

	stepLog := "has to schedule apps, stop volume driver on app nodes and destroy apps"

	It(stepLog, func() {
		log.InfoD(stepLog)
		contexts = make([]*scheduler.Context, 0)

		for i := 0; i < Inst().GlobalScaleFactor; i++ {
			contexts = append(contexts, ScheduleApplications(fmt.Sprintf("voldriverappdown-%d", i))...)
		}

		ValidateApplications(contexts)

		r := rand.New(rand.NewSource(time.Now().UnixNano()))

		stepLog = "get nodes for all apps in test and bounce volume driver"
		Step(stepLog, func() {
			log.InfoD(stepLog)
			for _, ctx := range contexts {
				appNodes, err := Inst().S.GetNodesForApp(ctx)
				log.FailOnError(err, "Failed to get nodes for the app %s", ctx.App.Key)
				appNode := appNodes[r.Intn(len(appNodes))]
				stepLog = fmt.Sprintf("stop volume driver %s on app %s's nodes: %v",
					Inst().V.String(), ctx.App.Key, appNode)
				Step(stepLog, func() {
					StopVolDriverAndWait([]node.Node{appNode})
				})

				stepLog = fmt.Sprintf("destroy app: %s", ctx.App.Key)
				Step(stepLog, func() {
					err = Inst().S.Destroy(ctx, nil)
					dash.VerifyFatal(err, nil, "Verify App delete")
					stepLog = "wait for few seconds for app destroy to trigger"
					Step(stepLog, func() {
						log.InfoD(stepLog)
						time.Sleep(10 * time.Second)
					})
				})

				stepLog = "restarting volume driver"
				Step(stepLog, func() {
					log.InfoD(stepLog)
					StartVolDriverAndWait([]node.Node{appNode})
				})

				stepLog = fmt.Sprintf("wait for destroy of app: %s", ctx.App.Key)
				Step(stepLog, func() {
					log.InfoD(stepLog)
					err = Inst().S.WaitForDestroy(ctx, Inst().DestroyAppTimeout)
					dash.VerifySafely(err, nil, fmt.Sprintf("Verify App %s deletion", ctx.App.Key))
				})

				DeleteVolumesAndWait(ctx, nil)
			}
		})
	})
	JustAfterEach(func() {
		defer EndTorpedoTest()
		AfterEachTest(contexts, testrailID, runID)
	})
})

// This test deletes all tasks of an application and checks if app converges back to desired state
var _ = Describe("{AppTasksDown}", func() {
	var testrailID = 35263
	// testrailID corresponds to: https://portworx.testrail.net/index.php?/cases/view/35264
	var runID int
	JustBeforeEach(func() {
		StartTorpedoTest("AppTasksDown", "Validate app after tasks are deleted", nil, testrailID)
		runID = testrailuttils.AddRunsToMilestone(testrailID)
	})
	var contexts []*scheduler.Context

	stepLog := "has to schedule app and delete app tasks"
	It(stepLog, func() {
		log.InfoD(stepLog)
		var err error
		contexts = make([]*scheduler.Context, 0)

		for i := 0; i < Inst().GlobalScaleFactor; i++ {
			contexts = append(contexts, ScheduleApplications(fmt.Sprintf("apptasksdown-%d", i))...)
		}

		ValidateApplications(contexts)

		stepLog = "delete all application tasks"
		Step(stepLog, func() {
			log.InfoD(stepLog)
			// Add interval based sleep here to check what time we will exit out of this delete task loop
			minRunTime := Inst().MinRunTimeMins
			timeout := (minRunTime) * 60
			// set frequency mins depending on the chaos level
			var frequency int
			switch Inst().ChaosLevel {
			case 5:
				frequency = 1
			case 4:
				frequency = 3
			case 3:
				frequency = 5
			case 2:
				frequency = 7
			case 1:
				frequency = 10
			default:
				frequency = 10

			}
			if minRunTime == 0 {
				for _, ctx := range contexts {
					stepLog = fmt.Sprintf("delete tasks for app: %s", ctx.App.Key)
					Step(stepLog, func() {
						err = Inst().S.DeleteTasks(ctx, nil)
						if err != nil {
							PrintDescribeContext(ctx)
						}
						dash.VerifyFatal(err, nil, fmt.Sprintf("validate delete tasks for app: %s", ctx.App.Key))
					})

					ValidateContext(ctx)
				}
			} else {
				start := time.Now().Local()
				for int(time.Since(start).Seconds()) < timeout {
					for _, ctx := range contexts {
						stepLog = fmt.Sprintf("delete tasks for app: %s", ctx.App.Key)
						Step(stepLog, func() {
							err = Inst().S.DeleteTasks(ctx, nil)
							if err != nil {
								PrintDescribeContext(ctx)
							}
							dash.VerifyFatal(err, nil, fmt.Sprintf("validate delete tasks for app: %s", ctx.App.Key))
						})

						ValidateContext(ctx)
					}
					stepLog = fmt.Sprintf("Sleeping for given duration %d", frequency)
					Step(stepLog, func() {
						log.InfoD(stepLog)
						d := time.Duration(frequency)
						time.Sleep(time.Minute * d)
					})
				}
			}
		})

		Step("teardown all apps", func() {
			for _, ctx := range contexts {
				TearDownContext(ctx, nil)
			}
		})
	})
	JustAfterEach(func() {
		defer EndTorpedoTest()
		AfterEachTest(contexts, testrailID, runID)
	})
})

// This test scales up and down an application and checks if app has actually scaled accordingly
var _ = Describe("{AppScaleUpAndDown}", func() {
	var testrailID = 35264
	// testrailID corresponds to: https://portworx.testrail.net/index.php?/cases/view/35264
	var runID int
	JustBeforeEach(func() {
		StartTorpedoTest("AppScaleUpAndDown", "Validate Apps sclae up and scale down", nil, testrailID)
		runID = testrailuttils.AddRunsToMilestone(testrailID)
	})
	var contexts []*scheduler.Context

	stepLog := "has to scale up and scale down the app"
	It(stepLog, func() {
		log.InfoD("has to scale up and scale down the app")
		contexts = make([]*scheduler.Context, 0)

		for i := 0; i < Inst().GlobalScaleFactor; i++ {
			contexts = append(contexts, ScheduleApplications(fmt.Sprintf("applicationscaleupdown-%d", i))...)
		}

		ValidateApplications(contexts)

		stepLog = "Scale up and down all app"
		Step(stepLog, func() {
			log.InfoD(stepLog)
			for _, ctx := range contexts {
				stepLog = fmt.Sprintf("scale up app: %s by %d ", ctx.App.Key, len(node.GetWorkerNodes()))
				Step(stepLog, func() {
					log.InfoD(stepLog)
					applicationScaleUpMap, err := Inst().S.GetScaleFactorMap(ctx)
					log.FailOnError(err, "Failed to get application scale up factor map")
					//Scaling up by number of storage-nodes
					workerStorageNodes := int32(len(node.GetStorageNodes()))
					for name, scale := range applicationScaleUpMap {
						// limit scale up to the number of worker nodes
						if scale < workerStorageNodes {
							applicationScaleUpMap[name] = workerStorageNodes
						}
					}
					err = Inst().S.ScaleApplication(ctx, applicationScaleUpMap)
					dash.VerifyFatal(err, nil, "Validate application scale up")
				})

				stepLog = "Giving few seconds for scaled up applications to stabilize"
				Step(stepLog, func() {
					log.InfoD(stepLog)
					time.Sleep(10 * time.Second)
				})

				ValidateContext(ctx)

				stepLog = fmt.Sprintf("scale down app %s by 1", ctx.App.Key)
				Step(stepLog, func() {
					log.InfoD(stepLog)
					applicationScaleDownMap, err := Inst().S.GetScaleFactorMap(ctx)
					log.FailOnError(err, "Failed to get application scale down factor map")

					for name, scale := range applicationScaleDownMap {
						applicationScaleDownMap[name] = scale - 1
					}
					err = Inst().S.ScaleApplication(ctx, applicationScaleDownMap)
					dash.VerifyFatal(err, nil, "Validate application scale down")
				})

				stepLog = "Giving few seconds for scaled up applications to stabilize"
				Step(stepLog, func() {
					log.InfoD(stepLog)
					time.Sleep(10 * time.Second)
				})

				ValidateContext(ctx)
			}
		})

		Step("teardown all apps", func() {
			for _, ctx := range contexts {
				TearDownContext(ctx, nil)
			}
		})

	})
	JustAfterEach(func() {
		defer EndTorpedoTest()
		AfterEachTest(contexts, testrailID, runID)
	})
})

var _ = Describe("{CordonDeployDestroy}", func() {
	var testrailID = 54373
	// testrailID corresponds to: https://portworx.testrail.net/index.php?/cases/view/54373
	var runID int
	JustBeforeEach(func() {
		StartTorpedoTest("CordonDeployDestroy", "Validate Cordon node and destroy app", nil, testrailID)
		runID = testrailuttils.AddRunsToMilestone(testrailID)
	})

	var contexts []*scheduler.Context

	stepLog := "has to cordon all nodes but one, deploy and destroy app"
	It(stepLog, func() {
		log.InfoD(stepLog)
		stepLog = "Cordon all nodes but one"

		Step(stepLog, func() {
			log.InfoD(stepLog)
			nodes := node.GetWorkerNodes()
			for _, node := range nodes[1:] {
				err := Inst().S.DisableSchedulingOnNode(node)
				dash.VerifyFatal(err, nil, fmt.Sprintf("Validate disable scheduling on node %s", node.Name))

			}
		})
		stepLog = "Deploy applications"
		Step(stepLog, func() {
			log.InfoD(stepLog)
			contexts = make([]*scheduler.Context, 0)

			for i := 0; i < Inst().GlobalScaleFactor; i++ {
				contexts = append(contexts, ScheduleApplications(fmt.Sprintf("cordondeploydestroy-%d", i))...)
			}
			ValidateApplications(contexts)

		})
		stepLog = "Destroy apps"
		Step(stepLog, func() {
			log.InfoD(stepLog)
			opts := make(map[string]bool)
			opts[scheduler.OptionsWaitForDestroy] = false
			opts[scheduler.OptionsWaitForResourceLeakCleanup] = false
			for _, ctx := range contexts {
				err := Inst().S.Destroy(ctx, opts)
				dash.VerifyFatal(err, nil, fmt.Sprintf("Validate App %s detroy init", ctx.App.Key))
			}
		})
		Step("Validate destroy", func() {
			for _, ctx := range contexts {
				err := Inst().S.WaitForDestroy(ctx, Inst().DestroyAppTimeout)
				dash.VerifyFatal(err, nil, fmt.Sprintf("Validate App %s detroy", ctx.App.Key))

			}
		})
		Step("teardown all apps", func() {
			for _, ctx := range contexts {
				TearDownContext(ctx, nil)
			}
		})
		Step("Uncordon all nodes", func() {
			nodes := node.GetWorkerNodes()
			for _, node := range nodes {
				err := Inst().S.EnableSchedulingOnNode(node)
				dash.VerifyFatal(err, nil, fmt.Sprintf("Validate enable scheduling on node %s", node.Name))
			}
		})
	})
	JustAfterEach(func() {
		defer EndTorpedoTest()
		AfterEachTest(contexts, testrailID, runID)

	})
})

var _ = Describe("{CordonStorageNodesDeployDestroy}", func() {
	JustBeforeEach(func() {
		StartTorpedoTest("CordonStorageNodesDeployDestroy", "Validate Cordon storage node , deploy and destroy app", nil, 0)

	})
	var contexts []*scheduler.Context

	stepLog := "has to cordon all storage nodes, deploy and destroy app"
	It(stepLog, func() {
		log.InfoD(stepLog)
		stepLog = "Cordon all storage nodes"
		Step(stepLog, func() {
			log.InfoD(stepLog)
			nodes := node.GetNodes()
			storageNodes := node.GetStorageNodes()
			if len(nodes) == len(storageNodes) {
				stepLog = "No storageless nodes detected. Skipping.."
				log.Warn(stepLog)
				Skip(stepLog)
			}
			for _, n := range storageNodes {
				err := Inst().S.DisableSchedulingOnNode(n)
				dash.VerifyFatal(err, nil, fmt.Sprintf("Validate disable scheduling on node %s", n.Name))
			}
		})
		stepLog = "Deploy applications"
		Step(stepLog, func() {
			log.InfoD(stepLog)

			contexts = make([]*scheduler.Context, 0)

			for i := 0; i < Inst().GlobalScaleFactor; i++ {
				contexts = append(contexts, ScheduleApplications(fmt.Sprintf("cordondeploydestroy-%d", i))...)
			}
			ValidateApplications(contexts)

		})
		stepLog = "Destroy apps"
		Step(stepLog, func() {
			log.InfoD(stepLog)
			opts := make(map[string]bool)
			opts[scheduler.OptionsWaitForDestroy] = false
			opts[scheduler.OptionsWaitForResourceLeakCleanup] = false
			for _, ctx := range contexts {
				err := Inst().S.Destroy(ctx, opts)
				dash.VerifyFatal(err, nil, fmt.Sprintf("Validate App %s detroy init", ctx.App.Key))
			}
		})
		Step("Validate destroy", func() {
			for _, ctx := range contexts {
				err := Inst().S.WaitForDestroy(ctx, Inst().DestroyAppTimeout)
				dash.VerifyFatal(err, nil, fmt.Sprintf("Validate App %s detroy", ctx.App.Key))

			}
		})
		Step("teardown all apps", func() {
			for _, ctx := range contexts {
				TearDownContext(ctx, nil)
			}
		})
		Step("Uncordon all nodes", func() {
			nodes := node.GetWorkerNodes()
			for _, node := range nodes {
				err := Inst().S.EnableSchedulingOnNode(node)
				dash.VerifyFatal(err, nil, fmt.Sprintf("Validate enable scheduling on node %s", node.Name))
			}
		})
	})
	JustAfterEach(func() {
		defer EndTorpedoTest()
		AfterEachTest(contexts)
	})
})

var _ = Describe("{SecretsVaultFunctional}", func() {
	var testrailID, runID int
	var contexts []*scheduler.Context
	var provider string

	const (
		vaultSecretProvider        = "vault"
		vaultTransitSecretProvider = "vault-transit"
		portworxContainerName      = "portworx"
	)

	BeforeEach(func() {
		StartTorpedoTest("SecretsVaultFunctional", "Validate Secrets Vault", nil, 0)
		isOpBased, _ := Inst().V.IsOperatorBasedInstall()
		if !isOpBased {
			k8sApps := apps.Instance()
			daemonSets, err := k8sApps.ListDaemonSets("kube-system", metav1.ListOptions{
				LabelSelector: "name=portworx",
			})
			log.FailOnError(err, "Failed to get daemon sets list")
			dash.VerifyFatal(len(daemonSets) > 0, true, "Daemon sets returned?")
			dash.VerifyFatal(len(daemonSets[0].Spec.Template.Spec.Containers) > 0, true, "Daemon set container is not empty?")
			usingVault := false
			for _, container := range daemonSets[0].Spec.Template.Spec.Containers {
				if container.Name == portworxContainerName {
					for _, arg := range container.Args {
						if arg == vaultSecretProvider || arg == vaultTransitSecretProvider {
							usingVault = true
							provider = arg
						}
					}
				}
			}
			if !usingVault {
				skipLog := fmt.Sprintf("Skip test for not using %s or %s ", vaultSecretProvider, vaultTransitSecretProvider)
				log.Warn(skipLog)
				Skip(skipLog)
			}
		} else {
			spec, err := Inst().V.GetStorageCluster()
			log.FailOnError(err, "Failed to get storage cluster")
			if *spec.Spec.SecretsProvider != vaultSecretProvider &&
				*spec.Spec.SecretsProvider != vaultTransitSecretProvider {
				Skip(fmt.Sprintf("Skip test for not using %s or %s ", vaultSecretProvider, vaultTransitSecretProvider))
			}
			provider = *spec.Spec.SecretsProvider
		}
	})

	var _ = Describe("{RunSecretsLogin}", func() {
		testrailID = 82774
		// testrailID corresponds to: https://portworx.testrail.net/index.php?/cases/view/82774
		JustBeforeEach(func() {
			StartTorpedoTest("RunSecretsLogin", "Test secrets login for vaults", nil, 0)
			runID = testrailuttils.AddRunsToMilestone(testrailID)
		})

		stepLog := "has to run secrets login for vault or vault-transit"

		It(stepLog, func() {
			log.InfoD(stepLog)
			contexts = make([]*scheduler.Context, 0)
			n := node.GetWorkerNodes()[0]
			if provider == vaultTransitSecretProvider {
				// vault-transit login with `pxctl secrets vaulttransit login`
				provider = "vaulttransit"
			}
			err := Inst().V.RunSecretsLogin(n, provider)
			dash.VerifyFatal(err, nil, "Validate secrets login")
		})
	})

	AfterEach(func() {
		defer EndTorpedoTest()
		AfterEachTest(contexts, testrailID, runID)
	})
})

var _ = Describe("{VolumeCreatePXRestart}", func() {
	JustBeforeEach(func() {
		StartTorpedoTest("VolumeCreatePXRestart", "Validate restart PX while create and attach", nil, 0)

	})
	contexts := make([]*scheduler.Context, 0)

	stepLog := "Validate volume attachment when px is restarting"
	It(stepLog, func() {
		var createdVolIDs map[string]string
		var err error
		volCreateCount := 10
		stepLog := "Create multiple volumes , attached and restart PX"
		Step(stepLog, func() {
			log.InfoD(stepLog)

			stNodes := node.GetStorageNodes()
			index := rand.Intn(len(stNodes))
			selectedNode := stNodes[index]

			log.InfoD("Creating and attaching %d volumes on node %s", volCreateCount, selectedNode.Name)

			wg := new(sync.WaitGroup)
			wg.Add(1)
			go func(appNode node.Node) {
				createdVolIDs, err = CreateMultiVolumesAndAttach(wg, volCreateCount, selectedNode.Id)
				if err != nil {
					log.Fatalf("Error while creating volumes. Err: %v", err)
				}
			}(selectedNode)
			time.Sleep(2 * time.Second)
			wg.Add(1)
			go func(appNode node.Node) {
				defer wg.Done()
				stepLog = fmt.Sprintf("restart volume driver %s on node: %s", Inst().V.String(), appNode.Name)
				Step(stepLog, func() {
					log.InfoD(stepLog)
					err = Inst().V.RestartDriver(appNode, nil)
					log.FailOnError(err, "Error while restarting volume driver")

				})
			}(selectedNode)
			wg.Wait()

		})

		stepLog = "Validate the created volumes"
		Step(stepLog, func() {
			log.InfoD(stepLog)

			for vol, volPath := range createdVolIDs {
				cVol, err := Inst().V.InspectVolume(vol)
				if err == nil {
					dash.VerifySafely(cVol.State, opsapi.VolumeState_VOLUME_STATE_ATTACHED, fmt.Sprintf("Verify vol %s is attached", cVol.Id))
					dash.VerifySafely(cVol.DevicePath, volPath, fmt.Sprintf("Verify vol %s is has device path", cVol.Id))
				} else {
					log.Fatalf("Error while inspecting volume %s. Err: %v", vol, err)
				}
			}
		})

		stepLog = "Deleting the created volumes"
		Step(stepLog, func() {
			log.InfoD(stepLog)

			for vol := range createdVolIDs {
				log.Infof("Detaching and deleting volume: %s", vol)
				err := Inst().V.DetachVolume(vol)
				if err == nil {
					err = Inst().V.DeleteVolume(vol)
				}
				log.FailOnError(err, "Error while deleting volume %s", vol)

			}
		})
	})

	JustAfterEach(func() {
		defer EndTorpedoTest()
		AfterEachTest(contexts)
	})
})
