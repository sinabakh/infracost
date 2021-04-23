#!/bin/sh -le

# This is an Jenkins-specific script that runs infracost on the current branch.
# Usage docs: https://www.infracost.io/docs/integrations/

fix_env_vars () {
    # Jenkins has problems with envs case sensivity
    iac_path=${iac_path:-$IAC_PATH}
    terraform_plan_flags=${terraform_plan_flags:-$TERRAFORM_PLAN_FLAGS}
    terraform_workspace=${terraform_workspace:-$TERRAFORM_WORKSPACE}
    usage_file=${usage_file:-$USAGE_FILE}
    config_file=${config_file:-$CONFIG_FILE}
    fail_condition=${fail_condition:-$FAIL_CONDITION}
}

process_args () {
  # Set variables based on the order for GitHub Actions, or the env value for other CIs
  iac_path=${1:-$iac_path}
  terraform_plan_flags=${2:-$terraform_plan_flags}
  terraform_workspace=${3:-$terraform_workspace}
  usage_file=${4:-$usage_file}
  config_file=${5:-$config_file}
  fail_condition=${7:-$fail_condition}

  # Validate fail_condition
  if ! echo "$fail_condition" | jq empty; then
    echo "Error: fail_condition contains invalid JSON"
  fi

  # Set defaults
  if [ ! -z "$fail_condition" ] && [ "$(echo "$fail_condition" | jq '.percentage_threshold')" != "null" ]; then
    fail_percentage_threshold=$(echo "$fail_condition" | jq -r '.percentage_threshold')
  fi
  fail_percentage_threshold=${fail_percentage_threshold:--1}
  INFRACOST_BINARY=${INFRACOST_BINARY:-infracost}

  # Export as it's used by infracost, not this script
  export INFRACOST_LOG_LEVEL=${INFRACOST_LOG_LEVEL:-info}
  export INFRACOST_CI_DIFF=true

  if [ ! -z "$GIT_SSH_KEY" ]; then
    echo "Setting up private Git SSH key so terraform can access your private modules."
    mkdir -p .ssh
    echo "${GIT_SSH_KEY}" > .ssh/git_ssh_key
    chmod 600 .ssh/git_ssh_key
    export GIT_SSH_COMMAND="ssh -i $(pwd)/.ssh/git_ssh_key -o 'StrictHostKeyChecking=no'"
  fi

}

build_breakdown_cmd () {
  breakdown_cmd="${INFRACOST_BINARY} breakdown --no-color --format json"

  if [ ! -z "$iac_path" ]; then
    breakdown_cmd="$breakdown_cmd --path $iac_path"
  fi
  if [ ! -z "$terraform_plan_flags" ]; then
    breakdown_cmd="$breakdown_cmd --terraform-plan-flags \"$terraform_plan_flags\""
  fi
  if [ ! -z "$usage_file" ]; then
    breakdown_cmd="$breakdown_cmd --usage-file $usage_file"
  fi
  if [ ! -z "$config_file" ]; then
    breakdown_cmd="$breakdown_cmd --config-file $config_file"
  fi
  echo "$breakdown_cmd"
}

build_output_cmd () {
  breakdown_path=$1
  output_cmd="${INFRACOST_BINARY} output --no-color --format diff --path $1"
  echo "${output_cmd}"
}

format_cost () {
  cost=$1
    
  if [ -z "$cost" ] | [ "${cost}" == "null" ]; then
    echo "-"
  elif [ $(echo "$cost < 100" | bc -l) = 1 ]; then
    printf "$%0.2f" $cost
  else
    printf "$%0.0f" $cost
  fi
}

build_msg () {
  change_word="increase"
  change_sym="+"
  if [ $(echo "$total_monthly_cost < ${past_total_monthly_cost}" | bc -l) = 1 ]; then
    change_word="decrease"
    change_sym=""
  fi

  percent_display=""
  if [ ! -z "$percent" ]; then
    percent_display="$(printf "%.0f" $percent)"
    percent_display=" (${change_sym}${percent_display}%%)"
  fi

  msg="${msg}\n##### Infracost estimate #####"
  msg="${msg}\n\n"
  msg="${msg}Monthly cost will ${change_word} by $(format_cost $diff_cost)$percent_display\n"
  msg="${msg}\n"
  msg="${msg}Previous monthly cost: $(format_cost $past_total_monthly_cost)\n"
  msg="${msg}New monthly cost: $(format_cost $total_monthly_cost)\n"
  msg="${msg}\n"
  msg="${msg}Infracost output:\n"
  msg="${msg}\n"
  msg="${msg}$(echo "    ${diff_output//$'\n'/\\n    }" | sed "s/%/%%/g")\n"
  printf "$msg"
}

build_msg_html () {
    msg=$1
    html="<!DOCTYPE html>\n<html>\n<body>\n<pre>"
    html="${html}\n${msg}"
    html="${html}</pre>\n</body>\n</html>"
    printf "$html"
}

cleanup () {
  rm -f infracost_breakdown.json infracost_breakdown_cmd infracost_output_cmd
}


# MAIN

fix_env_vars
process_args "$@"

infracost_breakdown_cmd=$(build_breakdown_cmd)
echo "$infracost_breakdown_cmd" > infracost_breakdown_cmd

echo "Running infracost breakdown using:"
echo "  $ $(cat infracost_breakdown_cmd)"
breakdown_output=$(cat infracost_breakdown_cmd | sh)
echo "$breakdown_output" > infracost_breakdown.json

infracost_output_cmd=$(build_output_cmd "infracost_breakdown.json")
echo "$infracost_output_cmd" > infracost_output_cmd
  
echo "Running infracost output using:"
echo "  $ $(cat infracost_output_cmd)"
diff_output=$(cat infracost_output_cmd | sh)

past_total_monthly_cost=$(jq '[.projects[].pastBreakdown.totalMonthlyCost | select (.!=null) | tonumber] | add' infracost_breakdown.json)
total_monthly_cost=$(jq '[.projects[].breakdown.totalMonthlyCost | select (.!=null) | tonumber] | add' infracost_breakdown.json)
diff_cost=$(jq '[.projects[].diff.totalMonthlyCost | select (.!=null) | tonumber] | add' infracost_breakdown.json)

# If both old and new costs are greater than 0
if [ $(echo "$past_total_monthly_cost > 0" | bc -l) = 1 ] && [ $(echo "$total_monthly_cost > 0" | bc -l) = 1 ]; then
  percent=$(echo "scale=6; $total_monthly_cost / $past_total_monthly_cost * 100 - 100" | bc)
fi

# If both old and new costs are less than or equal to 0
if [ $(echo "$past_total_monthly_cost <= 0" | bc -l) = 1 ] && [ $(echo "$total_monthly_cost <= 0" | bc -l) = 1 ]; then
  percent=0
fi

absolute_percent=$(echo $percent | tr -d -)
diff_resources=$(jq '[.projects[].diff.resources[]] | add' infracost_breakdown.json)

is_failure=0
if [ -z $percent ]; then
  echo "Passing as as percentage diff is empty."
elif [ $(echo "$fail_percentage_threshold < 0" | bc -l) = 1 ]; then
  echo "Passing as no fail percentage threshold is specified."  
elif [ $(echo "$absolute_percent > $fail_percentage_threshold" | bc -l) = 1 ]; then
  echo "Failing as percentage diff ($absolute_percent%) is greater than the percentage threshold ($fail_percentage_threshold%)."
  is_failure=1
else
  echo "Passing as percentage diff ($absolute_percent%) is less than or equal to percentage threshold ($fail_percentage_threshold%)."
fi

msg="$(build_msg)"
echo "$msg"

html=$(build_msg_html "$msg")
echo "$html" > infracost_diff_output.html

cleanup

exit $is_failure
