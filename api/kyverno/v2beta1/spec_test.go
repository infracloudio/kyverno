package v2beta1

import (
	"testing"

	kyvernov1 "github.com/kyverno/kyverno/api/kyverno/v1"
	"gotest.tools/assert"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func Test_Validate_UniqueRuleName(t *testing.T) {
	subject := Spec{
		Rules: []Rule{{
			Name: "deny-privileged-disallowpriviligedescalation",
			MatchResources: MatchResources{
				Any: kyvernov1.ResourceFilters{{
					ResourceDescription: kyvernov1.ResourceDescription{
						Kinds: []string{
							"Pod",
						},
					},
				}},
			},
			Validation: Validation{
				Message: "message",
				RawAnyPattern: &apiextv1.JSON{
					Raw: []byte("{"),
				},
			},
		}, {
			Name: "deny-privileged-disallowpriviligedescalation",
			MatchResources: MatchResources{
				Any: kyvernov1.ResourceFilters{{
					ResourceDescription: kyvernov1.ResourceDescription{
						Kinds: []string{
							"Pod",
						},
					}},
				}},
			Validation: Validation{
				Message: "message",
				RawAnyPattern: &apiextv1.JSON{
					Raw: []byte("{"),
				},
			},
		}},
	}
	path := field.NewPath("dummy")
	errs := subject.Validate(path, false, nil)
	assert.Equal(t, len(errs), 1)
	assert.Equal(t, errs[0].Field, "dummy.rules[1].name")
	assert.Equal(t, errs[0].Type, field.ErrorTypeInvalid)
	assert.Equal(t, errs[0].Detail, "Duplicate rule name: 'deny-privileged-disallowpriviligedescalation'")
}

func Test_Validate_Namespaces(t *testing.T) {
	path := field.NewPath("dummy")
	testcases := []struct {
		description        string
		spec               Spec
		expectedSpecErrIdx int //variable to get the index from the Spec.ValidationFailureActionOverrides field
		errors             func(v *kyvernov1.ValidationFailureActionOverride) field.ErrorList
	}{
		{
			description: "Duplicate Namespace in 2nd resource(validation)",
			spec: Spec{
				ValidationFailureAction: kyvernov1.Enforce,
				ValidationFailureActionOverrides: []kyvernov1.ValidationFailureActionOverride{
					{
						Action: kyvernov1.Enforce,
						Namespaces: []string{
							"default",
							"test",
						},
					},
					{
						Action: kyvernov1.Audit,
						Namespaces: []string{
							"default",
						},
					},
				},
				Rules: []Rule{
					{
						Name: "require-labels",
						MatchResources: MatchResources{
							Any: kyvernov1.ResourceFilters{{
								ResourceDescription: kyvernov1.ResourceDescription{
									Kinds: []string{
										"Pod",
									},
								},
							}},
						},
						Validation: Validation{
							Message: "label 'app.kubernetes.io/name' is required",
							RawPattern: &apiextv1.JSON{
								Raw: []byte(`
							"metadata": {
								"lables": {
									"app.kubernetes.io/name": "?*"
								}
							}`),
							},
						},
					},
				},
			},
			expectedSpecErrIdx: 1,
			errors: func(v *kyvernov1.ValidationFailureActionOverride) (errs field.ErrorList) {
				return append(errs, field.Invalid(field.NewPath("dummy.validationFailureActionOverrides[1].namespaces"), v, "Duplicate namespaces found: default"))
			},
		},
		{
			description: "Duplicate Namespace in 2nd resource(mutate)",
			spec: Spec{
				ValidationFailureAction: kyvernov1.Enforce,
				ValidationFailureActionOverrides: []kyvernov1.ValidationFailureActionOverride{
					{
						Action: kyvernov1.Enforce,
						Namespaces: []string{
							"default",
							"test",
						},
					},
					{
						Action: kyvernov1.Audit,
						Namespaces: []string{
							"default",
						},
					},
				},
				Rules: []Rule{
					{
						Name: "add-labels",
						MatchResources: MatchResources{
							Any: kyvernov1.ResourceFilters{{
								ResourceDescription: kyvernov1.ResourceDescription{
									Kinds: []string{
										"Pod",
									},
								},
							}},
						},
						Mutation: kyvernov1.Mutation{
							RawPatchStrategicMerge: &apiextv1.JSON{
								Raw: []byte(`
								"metadata": {
									"labels": {
										"app-name": "{{request.object.metadata.name}}"
									}
								}`),
							},
						},
					},
				},
			},
			expectedSpecErrIdx: 1,
			errors: func(v *kyvernov1.ValidationFailureActionOverride) (errs field.ErrorList) {
				return append(errs, field.Invalid(field.NewPath("dummy.validationFailureActionOverrides[1].namespaces"), v, "Duplicate namespaces found: default"))
			},
		},
	}

	for _, ts := range testcases {
		t.Run(ts.description, func(t *testing.T) {
			errs := ts.spec.Validate(path, false, nil)

			var expectedErrs field.ErrorList
			if ts.errors != nil {
				expectedErrs = ts.errors(&ts.spec.ValidationFailureActionOverrides[ts.expectedSpecErrIdx])
			}

			assert.Equal(t, len(errs), len(expectedErrs))
			for i := range errs {
				assert.Equal(t, errs[i].Error(), expectedErrs[i].Error())
			}
		})
	}
}
