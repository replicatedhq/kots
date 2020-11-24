package identity

import (
	extensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var crdMeta = metav1.TypeMeta{
	APIVersion: "apiextensions.k8s.io/v1beta1",
	Kind:       "CustomResourceDefinition",
}

const dexApiGroup = "dex.coreos.com"

var DexCustomResourceDefinitions = []extensionsv1beta1.CustomResourceDefinition{
	{
		ObjectMeta: metav1.ObjectMeta{
			Name: "authcodes.dex.coreos.com",
		},
		TypeMeta: crdMeta,
		Spec: extensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   dexApiGroup,
			Version: "v1",
			Names: extensionsv1beta1.CustomResourceDefinitionNames{
				Plural:   "authcodes",
				Singular: "authcode",
				Kind:     "AuthCode",
			},
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name: "authrequests.dex.coreos.com",
		},
		TypeMeta: crdMeta,
		Spec: extensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   dexApiGroup,
			Version: "v1",
			Names: extensionsv1beta1.CustomResourceDefinitionNames{
				Plural:   "authrequests",
				Singular: "authrequest",
				Kind:     "AuthRequest",
			},
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name: "oauth2clients.dex.coreos.com",
		},
		TypeMeta: crdMeta,
		Spec: extensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   dexApiGroup,
			Version: "v1",
			Names: extensionsv1beta1.CustomResourceDefinitionNames{
				Plural:   "oauth2clients",
				Singular: "oauth2client",
				Kind:     "OAuth2Client",
			},
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name: "signingkeies.dex.coreos.com",
		},
		TypeMeta: crdMeta,
		Spec: extensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   dexApiGroup,
			Version: "v1",
			Names: extensionsv1beta1.CustomResourceDefinitionNames{
				// `signingkeies` is an artifact from the old TPR pluralization.
				// Users don't directly interact with this value, hence leaving it
				// as is.
				Plural:   "signingkeies",
				Singular: "signingkey",
				Kind:     "SigningKey",
			},
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name: "refreshtokens.dex.coreos.com",
		},
		TypeMeta: crdMeta,
		Spec: extensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   dexApiGroup,
			Version: "v1",
			Names: extensionsv1beta1.CustomResourceDefinitionNames{
				Plural:   "refreshtokens",
				Singular: "refreshtoken",
				Kind:     "RefreshToken",
			},
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name: "passwords.dex.coreos.com",
		},
		TypeMeta: crdMeta,
		Spec: extensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   dexApiGroup,
			Version: "v1",
			Names: extensionsv1beta1.CustomResourceDefinitionNames{
				Plural:   "passwords",
				Singular: "password",
				Kind:     "Password",
			},
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name: "offlinesessionses.dex.coreos.com",
		},
		TypeMeta: crdMeta,
		Spec: extensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   dexApiGroup,
			Version: "v1",
			Names: extensionsv1beta1.CustomResourceDefinitionNames{
				Plural:   "offlinesessionses",
				Singular: "offlinesessions",
				Kind:     "OfflineSessions",
			},
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name: "connectors.dex.coreos.com",
		},
		TypeMeta: crdMeta,
		Spec: extensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   dexApiGroup,
			Version: "v1",
			Names: extensionsv1beta1.CustomResourceDefinitionNames{
				Plural:   "connectors",
				Singular: "connector",
				Kind:     "Connector",
			},
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name: "devicerequests.dex.coreos.com",
		},
		TypeMeta: crdMeta,
		Spec: extensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   dexApiGroup,
			Version: "v1",
			Names: extensionsv1beta1.CustomResourceDefinitionNames{
				Plural:   "devicerequests",
				Singular: "devicerequest",
				Kind:     "DeviceRequest",
			},
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name: "devicetokens.dex.coreos.com",
		},
		TypeMeta: crdMeta,
		Spec: extensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   dexApiGroup,
			Version: "v1",
			Names: extensionsv1beta1.CustomResourceDefinitionNames{
				Plural:   "devicetokens",
				Singular: "devicetoken",
				Kind:     "DeviceToken",
			},
		},
	},
}
