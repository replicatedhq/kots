%% build with docker run --rm -u $(id -u):$(id -g) -v $PWD:/data minlag/mermaid-cli:9.4.0 -i ./update_flow.mms -o update_flow.svg

%% Use the newest renderer
%%{init: {"flowchart": {"defaultRenderer": "elk"}} }%%
%%{init: {"flowchart": {"defaultRenderer": "dagre"}} }%%

flowchart TB
	request(Update Request) --> helm_managed1?{Helm managed?}
	helm_managed1? --> |Yes| get_helm_app(Get helm app from in-memory cache)
	helm_managed1? --> |No| get_app1([Get app from store using app slug])

	get_helm_app --> request_type?{Requst type?}
	get_app1 --> request_type?
	request_type? --> |Multipart Form Data| todo1(TODO: airgap update)
	request_type? --> |JSON| update_status([Get update-download job status from store])

	update_status --> job_running?{update-download job running?}
	job_running? --> |Yes| done1(update in progress, nothing to do)
	job_running? --> |No| set_job_state([Set update-download job state to running in store])

	set_job_state --> helm_managed2?{Helm managed?}
	helm_managed2? --> |Yes| todo2(TODO: helm managed)
	helm_managed2? --> |No| get_app2([Get app from store])

	get_app2 --> get_lic_seq([Get latest app sequence and license from store])

	subgraph "Synchronize License"
		get_lic_seq --> request_license((Request latest license from replicated API))

		request_license --> get_app3([Get app archive from store]) --> load_kinds1(Load KOTS Kinds from app archive)

		load_kinds1 --> license_synced?{Current and latest icense data match?}
		license_synced? --> |No| write_license([Write new license data to store])
		license_synced? --> |Yes| mark_updated([Mark license synced in store])
	end

	write_license --> get_app4([Get app from store using ID in case a new sequence was generated])
	mark_updated --> get_app4
	get_app4 --> get_cursor([Get current update cursor from store])
	get_cursor --> get_updates((Get latest release updates from replicated API '/release/$APP_SLUG/pending' using latest license and update cursor))
	get_updates --> get_downstreams1([Get app downstream from store]) --> filter_updates(Filter out old updates)

	filter_updates --> any_updates?{Any updates?}
	any_updates? --> |No| set_last_update[(Write last update time to DB)] --> todo3(TODO: ensure correct version deployed)
	any_updates? --> |Yes| set_job_state2([Set update-download job state to running])

	set_job_state2 --> more_updates?{More updates?}
	more_updates? --> |Yes| set_job_state3([Set update-download.app_sequence job state to running])
	set_job_state3 --> extract_archive([Get and extract the app archive to temp dir]) --> load_kinds2(Load KOTS KINDS from archive)
	load_kinds2 --> get_app5([Get app, downstreams, namespace, next app sequence, latest license, and registry settings from store using ID])
	get_app5 --> clean_archive([Delete everything except from upstream and overlay from archive])
	clean_archive --> build_pull_options([Build pull options from data])

	build_pull_options --> verify_license(Verify the license signature) --> load_config_vals(Load config values file)
	load_config_vals --> load_identity_config(Load identity config file) --> load_installation_file(Load installation file)

	load_installation_file --> is_airgap?{Is airgap installation?}
	is_airgap? --> |Yes| todo4(TODO: airgap update)
	is_airgap? --> |No| download_new_app((Download newest app from replicated vendor))

	subgraph "Write upstreams"
		download_new_app --> create_upstream_dir(Create upstream dir)
		create_upstream_dir --> include_admin?{Include admin console?}
		include_admin? --> |Yes| generate_admin_manifests(Generate KOTS admin console manifests)
		include_admin? --> |No| load_previous_installation_data(Load previous installation encryption key from disk)
		generate_admin_manifests --> load_previous_installation_data
		load_previous_installation_data --> write_upstream_files(Write all upstream file contents, possibly encrypting the config and identity file data)
		write_upstream_files --> create_userdata_dir(Create the userdata dir if missing)
		create_userdata_dir --> generate_installation(Generate and write installation.yaml for upstream to userdata)
	end

	generate_installation --> load_kinds3(Load KOTS kinds from upstream files)
	load_kinds3 --> check_configuration(Check for KOTS configuration values that are required and unset)

	check_configuration --> is_airgap_config?{Is airgap, needs config, and rewrite images?}
	is_airgap_config? --> |Yes| todo5(TODO: airgap image rewrite)
	is_airgap_config? --> |No| load_kots_helm(Load any KOTS HelmChart manifests in processed files)
	todo5 --> load_kots_helm
	load_kots_helm --> check_install_method(Check that the helm installation method hasn't changed for non-excluded charts)

	check_install_method --> is_helm_or_repli{Upstream type is Helm or Replicated?}
	subgraph "Render Upstream"
		is_helm_or_repli --> |Helm| todo6(TODO: Helm type rendering)

		is_helm_or_repli --> |Replicated| load_kots_kinds(Load KOTS kinds from in-memory upstream files)
		load_kots_kinds --> create_config_builder(Create the config template builder with the KOTS kinds, upstream, and render options)
		create_config_builder --> load_kots_kinds2(Load KOTS kinds from the upstream files)
		load_kots_kinds2 --> create_template_builder(Create a template builder for the config spec and values)
		create_template_builder --> apply_config_values(Apply the config values to the config spec) --> render_template(Render the replicated templates in the config spec)

		render_template --> for_each_upstream{For each upstream file}
		for_each_upstream -->  |More| exclude_kots_kinds?{Exclude KOTS kinds?}
			exclude_kots_kinds? --> |Yes| filter_kots_kinds(Filter KOTS kinds from in-memory upstream files)
			exclude_kots_kinds? --> |No| todo7(TODO: Render variadic config templating)
			filter_kots_kinds --> todo7
			todo7 --> render_upstream(Render replicated templating) --> split_base_file(Split the rendered base file into single docs)
			split_base_file --> check_for_exclusion(Unmarshal the file and check if it is a KOTS manifest)
			check_for_exclusion --> exclude_and_is_kots?{Exclude KOTS kinds and is KOTS kind?}
			exclude_and_is_kots? --> |No| include_in_base(Include rendered file in base files)
			exclude_and_is_kots? --> |Yes| for_each_upstream
			include_in_base --> for_each_upstream
		for_each_upstream ----> |Done| for_each_upstream2{For each upstream file}

		for_each_upstream2 --> |More| check_is_helm(Unmarshal yaml and check for KOTS HelmChart kind)
			check_is_helm --> is_helm_chart?{Is a KOTS HelmChart?}
			is_helm_chart? --> |No| for_each_upstream2
			is_helm_chart? --> |Yes| render_replicated_helm(Render the HelmChart replicated templating)
			render_replicated_helm --> decode_helm_chart(Decode helm chart with K8s universal deserializer)
			decode_helm_chart --> append_to_charts(Add chart data to list of charts)
			append_to_charts --> for_each_upstream2
		for_each_upstream2 --> |Done| for_each_helm_chart{For each helm chart}

		for_each_helm_chart --> |More| is_excluded_spec?{Is HelmChart Spec.Exclude true?}
			is_excluded_spec? --> |Yes| for_each_helm_chart
			is_excluded_spec? --> |No| for_each_upstream3{For each upstream file}
			for_each_upstream3 --> |More| is_tar_archive?{Is a tar archive?}
				is_tar_archive? --> |No| for_each_upstream3
				is_tar_archive? --> |Yes| create_temporary_dir(Create a temporary directory 'chart')
				create_temporary_dir --> copy_archive_to_tmp(Write the archive file to the temporary directory)
				copy_archive_to_tmp --> unarchive_to_mem(Extract the archive file to memory)
				unarchive_to_mem --> for_each_archive_file{For each archive file}
				for_each_archive_file --> |More| is_path_chart_yaml?{Is the path 'Chart.yaml'}
					is_path_chart_yaml? --> |No| for_each_archive_file
					is_path_chart_yaml? --> |Yes| unmarshal_chart_yaml(Unmarshal the yaml data)
					unmarshal_chart_yaml --> chart_name_matches_manifest?{Chart name matches manifest name?}
					chart_name_matches_manifest? --> |No| for_each_archive_file
					chart_name_matches_manifest? --> |Yes| create_temp_dir(Create temp directory 'kots')
				for_each_archive_file --> |Done| for_each_upstream3
			for_each_upstream3 --> |Done| error_no_helm(No helm charts found)

			create_temp_dir --> copy_helm_archive_to_tmp(Copy the helm archive to the 'kots' temp directory)
			copy_helm_archive_to_tmp --> extract_archive_to_mem(Extract archive files to memory)
			extract_archive_to_mem --> merge_helm_values(Filter out false optional values and merge remaining HelmChart Spec.Values) 
			merge_helm_values --> convert_helm_value_types(Convert the values to correct types)
			convert_helm_value_types --> create_temporary_dir2(Create temporary directory 'kots') --> write_helm_files(Write all helm directories and files to temporary directory)
			write_helm_files --> parse_helm_options(Parse all helm options into values ??)
			parse_helm_options --> helm_chart_version?{Helm Chart version?}
			helm_chart_version? --> |v2| todo8(TODO: render Helm v2)
			helm_chart_version? --> |v3| load_helm_chart(Load Helm files into Helm lib structs)
			load_helm_chart --> check_helm_dependencies(Check that all helm dependencies exist)
			check_helm_dependencies --> dry_run_helm_generate(Generate Helm release with dry run Helm client install)
			dry_run_helm_generate --> coalesce_values(Helm lib coalesce values in release chart and config ??)
			coalesce_values --> marshal_coalesced_values(Marshall coalesced values using K8s yaml)
			marshal_coalesced_values --> add_hooks_crds(Create and add manifests for hooks and custom resources)
			add_hooks_crds --> filter_manifests(Filter empty manifests and fix sources for multidoc manifests)
			filter_manifests --> use_native_helm?{Use native Helm?}
			use_native_helm? --> |No| remove_common_prefix(Remove common Path prefixes from manifests)
			remove_common_prefix --> create_helm_release_secret(Create manifest for secret containing Helm release named the release name)

			use_native_helm? --> |No| create_temp_dir2(Create a temporary directory 'chart')
			create_helm_release_secret --> create_temp_dir2
			create_temp_dir2 --> for_each_helm_file{For each Helm file}
			for_each_helm_file --> |More| is_manifest_empty_custom_or_namespace{Is manifest empty, namespace, or custom resource?}
				is_manifest_empty_custom_or_namespace --> |Yes| for_each_helm_file
				is_manifest_empty_custom_or_namespace --> |No| write_temp_manifest(Write manifest to temporary file)
				write_temp_manifest --> add_to_kustomize_path(Add temporary file path to kustomize path list)
				add_to_kustomize_path --> for_each_helm_file
			for_each_helm_file --> |Done| create_kustomize_yaml(Create, marshal, and write kustomize yaml from path list)
			create_kustomize_yaml --> use_krusty(Use Krusty to create and run the kustomizations)
			use_krusty --> merge_kustomization(Merge the kustomized yamls with non-kustomized)
			merge_kustomization --> dedup_manifests(De-duplicate manifests)

			dedup_manifests --> use_native_helm2?{Use native Helm?}
			todo8 --> use_native_helm2?
			use_native_helm2? --> |No| remove_common_prefixes{Remove common path prefixes from each rendered helm file}
			use_native_helm2? --> |Yes| for_each_helm_file2{For each helm file}
		
			remove_common_prefixes --> for_each_helm_file2
			for_each_helm_file2 --> |More| split_multidoc(Split multidoc files if option is set)
				split_multidoc --> decode_helm_file(Decode the file using K8s universal deserializer)
				decode_helm_file --> is_job_kind?{Is Job kind?}
				is_job_kind? --> |No| for_each_helm_file2
				is_job_kind? --> |Yes| rename_hook_delete(Rename hook-delete-policy annotation from 'helm.sh' to 'kots.io')
				rename_hook_delete --> serialize_modified(Serialize the modified manifest back to yaml)
				serialize_modified --> for_each_helm_file2
			for_each_helm_file2 --> |Done| combine_files(Combine modified and unmodified helm files)
			combine_files --> generate_file_maps(Generate file maps for base to upstream and upstream to base)
			generate_file_maps --> add_dependencies(Add back any additional or missing helm chart files and dependencies???)

			add_dependencies --> for_each_helm_base_file{For each file in the helm base}
			for_each_helm_base_file --> |More| render_replicated_template(Render the replicated templating)
				render_replicated_template --> unmarshal_helm_file(unmarshal file and check if there is a KOTS exclude annotation)
				unmarshal_helm_file --> for_each_base{For each Helm Base}
				for_each_base --> |More| for_each_helm_base_file
				for_each_base --> |Done| for_each_helm_base_file
			for_each_helm_base_file --> |Done| generate_new_base(Generate new base from included files)
			generate_new_base --> for_each_helm_chart
	end

		for_each_helm_chart --> |Done| create_and_write_base(Create base directory and write the processed base replicated manifests)
		todo6 --> create_and_write_base
		create_and_write_base --> for_each_helm_base{For each Helm base}
		for_each_helm_base --> |More| deep_copy_base(Make a deep copy of the base??)
			deep_copy_base --> remove_namespace(Set the namespace to blank)
			remove_namespace --> create_and_write_base2(Create any directories and write all processed base helm manifests)
			create_and_write_base2 --> for_each_helm_base

	for_each_helm_base ----> |Done| todo9(TODO: write midstreams)
	todo9 --> todo10(TODO: write downstreams)
	todo10 --> todo11(TODO: kustomize write rendered app)  --> update_app_version([Create or update app version in store])
	update_app_version --> run_preflights(Run preflights)

	success_response((Status OK response))
	done1 ----> success_response

	error_response((Error resposne))
	error_no_helm --> error_response

	%% Style overrides
	style success_response fill:#D6FFB7,stroke:#080357,stroke-width:4px
	style error_response fill:#FF9F1C,stroke:#080357,stroke-width:4px

	style todo1 fill:#F5FF90,stroke:#080357,stroke-width:2px
	style todo2 fill:#F5FF90,stroke:#080357,stroke-width:2px
	style todo3 fill:#F5FF90,stroke:#080357,stroke-width:2px
	style todo4 fill:#F5FF90,stroke:#080357,stroke-width:2px
	style todo5 fill:#F5FF90,stroke:#080357,stroke-width:2px
	style todo6 fill:#F5FF90,stroke:#080357,stroke-width:2px
	style todo7 fill:#F5FF90,stroke:#080357,stroke-width:2px
	style todo8 fill:#F5FF90,stroke:#080357,stroke-width:2px
