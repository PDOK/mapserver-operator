#!/usr/bin/env bash
set -eo pipefail
echo "creating legends for root and group layers by concatenating data layers"
input_filepath="/input/input"
remove_filepath="/input/remove"
config_filepath="/input/ogc-webservice-proxy-config.yaml"
legend_dir="/var/www/legend"
< "${input_filepath}" xargs -n 2 echo | while read -r layer style; do
  export layer
  # shellcheck disable=SC2016 # dollar is for yq
  if ! < "${config_filepath}" yq -e 'env(layer) as $layer | .grouplayers | keys | contains([$layer])' &>/dev/null; then
    continue
  fi
  export grouplayer="${layer}"
  grouplayer_style_filepath="${legend_dir}/${grouplayer}/${style}.png"
  # shellcheck disable=SC2016 # dollar is for yq
  datalayers=$(< "${config_filepath}" yq 'env(grouplayer) as $foo | .grouplayers[$foo][]')
  datalayer_style_filepaths=()
  for datalayer in $datalayers; do
    datalayer_style_filepath="${legend_dir}/${datalayer}/${style}.png"
    if [[ -f "${datalayer_style_filepath}" ]]; then
      datalayer_style_filepaths+=("${datalayer_style_filepath}")
    fi
  done
  if [[ -n "${datalayer_style_filepaths[*]}" ]]; then
    echo "concatenating ${grouplayer_style_filepath}"
    gm convert -append "${datalayer_style_filepaths[@]}" "${grouplayer_style_filepath}"
  else
    echo "no data for ${grouplayer_style_filepath}"
  fi
done
< "${remove_filepath}" xargs -n 2 echo | while read -r layer style; do
  remove_legend_file="${legend_dir}/${layer}/${style}.png"
  echo removing $remove_legend_file
  rm $remove_legend_file
done
echo "done"