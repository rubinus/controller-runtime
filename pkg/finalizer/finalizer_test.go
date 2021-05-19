package finalizer

import (
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
)

type mockFinalizer struct {
	needsUpdate bool
	err         error
}

func (f mockFinalizer) Finalize(context.Context, client.Object) (needsUpdate bool, err error) {
	return f.needsUpdate, f.err
}
func TestFinalizer(t *testing.T) {
	RegisterFailHandler(Fail)
	suiteName := "Finalizer Suite"
	RunSpecsWithDefaultAndCustomReporters(t, suiteName, []Reporter{printer.NewlineReporter{}, printer.NewProwReporter(suiteName)})
}

var _ = Describe("TestFinalizer", func() {
	var err error
	var pod *corev1.Pod
	var finalizers Finalizers
	var f mockFinalizer
	BeforeEach(func() {
		pod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Finalizers: []string{"finalizers.sigs.k8s.io/testfinalizer"},
			},
		}
		Expect(pod).NotTo(BeNil())

		finalizers = NewFinalizers()
		Expect(finalizers).NotTo(BeNil())

		f := mockFinalizer{}
		Expect(f).NotTo(BeNil())

	})
	Describe("Register", func() {
		It("successfully registers a finalizer", func() {
			err = finalizers.Register("finalizers.sigs.k8s.io/testfinalizer", f)
			Expect(err).To(BeNil())
		})

		It("should fail when trying to register a finalizer that was already registered", func() {
			err = finalizers.Register("finalizers.sigs.k8s.io/testfinalizer", f)
			Expect(err).To(BeNil())

			// calling Register again with the same key should return an error
			err = finalizers.Register("finalizers.sigs.k8s.io/testfinalizer", f)
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("already registered"))

		})
	})
	Describe("Finalize", func() {
		It("should return no error and return false for needsUpdate if a finalizer is not registered", func() {
			ret, err := finalizers.Finalize(context.TODO(), pod)
			Expect(err).To(BeNil())
			Expect(ret).To(BeFalse())
		})

		It("successfully finalizes and returns true for needsUpdate when deletion timestamp is nil and finalizer does not exist", func() {
			err = finalizers.Register("finalizers.sigs.k8s.io/testfinalizer", f)
			Expect(err).To(BeNil())

			pod.DeletionTimestamp = nil
			pod.Finalizers = []string{}

			needsUpdate, err := finalizers.Finalize(context.TODO(), pod)
			Expect(err).To(BeNil())
			Expect(needsUpdate).To(BeTrue())
		})

		It("successfully finalizes and returns true for needsUpdate when deletion timestamp is not nil and the finalizer exists", func() {
			now := metav1.Now()
			pod.DeletionTimestamp = &now

			err = finalizers.Register("finalizers.sigs.k8s.io/testfinalizer", f)
			Expect(err).To(BeNil())

			pod.Finalizers = []string{"finalizers.sigs.k8s.io/testfinalizer"}

			needsUpdate, err := finalizers.Finalize(context.TODO(), pod)
			Expect(err).To(BeNil())
			Expect(needsUpdate).To(BeTrue())
		})

		It("should return no error and return false for needsUpdate when deletion timestamp is nil and finalizer doesn't exist", func() {
			pod.DeletionTimestamp = nil
			pod.Finalizers = []string{}

			needsUpdate, err := finalizers.Finalize(context.TODO(), pod)
			Expect(err).To(BeNil())
			Expect(needsUpdate).To(BeFalse())
		})

		It("should return no error and return false for needsUpdate when deletion timestamp is not nil and the finalizer doesn't exist", func() {
			now := metav1.Now()
			pod.DeletionTimestamp = &now
			pod.Finalizers = []string{}

			needsUpdate, err := finalizers.Finalize(context.TODO(), pod)
			Expect(err).To(BeNil())
			Expect(needsUpdate).To(BeFalse())

		})

		It("successfully finalizes multiple finalizers and returns true for needsUpdate when deletion timestamp is not nil and the finalizer exists", func() {
			now := metav1.Now()
			pod.DeletionTimestamp = &now

			err = finalizers.Register("finalizers.sigs.k8s.io/testfinalizer", f)
			Expect(err).To(BeNil())

			err = finalizers.Register("finalizers.sigs.k8s.io/newtestfinalizer", f)
			Expect(err).To(BeNil())

			pod.Finalizers = []string{"finalizers.sigs.k8s.io/testfinalizer", "finalizers.sigs.k8s.io/newtestfinalizer"}

			needsUpdate, err := finalizers.Finalize(context.TODO(), pod)
			Expect(err).To(BeNil())
			Expect(needsUpdate).To(BeTrue())
		})

		It("should return needsUpdate as false and a non-nil error", func() {
			now := metav1.Now()
			pod.DeletionTimestamp = &now
			pod.Finalizers = []string{"finalizers.sigs.k8s.io/testfinalizer"}

			f.needsUpdate = false
			f.err = fmt.Errorf("finalizer failed for %q", pod.Finalizers[0])

			err = finalizers.Register("finalizers.sigs.k8s.io/testfinalizer", f)
			Expect(err).To(BeNil())

			needsUpdate, err := finalizers.Finalize(context.TODO(), pod)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("finalizer failed"))
			Expect(needsUpdate).To(BeFalse())
		})

		It("should return expected needsUpdate and error values when registering multiple finalizers", func() {
			now := metav1.Now()
			pod.DeletionTimestamp = &now
			pod.Finalizers = []string{
				"finalizers.sigs.k8s.io/testfinalizer1",
				"finalizers.sigs.k8s.io/testfinalizer2",
				"finalizers.sigs.k8s.io/testfinalizer3",
			}

			// registering multiple finalizers with different return values
			// test for needsUpdate as true, and nil error
			f.needsUpdate = true
			f.err = nil
			err = finalizers.Register("finalizers.sigs.k8s.io/testfinalizer1", f)
			Expect(err).To(BeNil())

			result, err := finalizers.Finalize(context.TODO(), pod)
			Expect(err).To(BeNil())
			Expect(result).To(BeTrue())

			// test for needsUpdate as false, and non-nil error
			f.needsUpdate = false
			f.err = fmt.Errorf("finalizer failed")
			err = finalizers.Register("finalizers.sigs.k8s.io/testfinalizer2", f)
			Expect(err).To(BeNil())

			result, err = finalizers.Finalize(context.TODO(), pod)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("finalizer failed"))
			Expect(result).To(BeFalse())

			// test for needsUpdate as true, and non-nil error
			f.needsUpdate = true
			f.err = fmt.Errorf("finalizer failed")
			err = finalizers.Register("finalizers.sigs.k8s.io/testfinalizer3", f)
			Expect(err).To(BeNil())

			result, err = finalizers.Finalize(context.TODO(), pod)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("finalizer failed"))
			Expect(result).To(BeTrue())
		})
	})
})
