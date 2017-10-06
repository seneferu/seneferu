function getRepos() {
    $.ajax({
        url: '/repos',
        type: 'GET',
        success: function (repos) {
            $('#repos-list').html('')

            repos.forEach(function (repo) {
                var repoLine = $("<div><a href='#'>" + repo.name + "</a></div>")
                repo.builds = repo.builds.reverse();
                repoLine.click(getRepoData.bind(null, repo));
                $('#repos-list').append(repoLine);
            });
        },
        error: function (err) {
            console.error(err)
        }
    });
}

function getBuildData(id, buildid) {
    $.ajax({
        url: '/repo/' + id + '/build/' + buildid,
        type: 'GET',
        success: function (buildData) {
            handleBuildClick(buildData)
        },
        error: function (err) {
            console.error(err)
        }
    });
}

function getRepoData(repo) {
    $.ajax({
        url: '/repo/' + repo.name,
        type: 'GET',
        success: function (data) {
            data.builds = data.builds.reverse();
            handleRepoClick(data);
        },
        error: function (err) {
            console.error(err)
        }
    });
}

function handleRepoClick(repoData) {
    $('#build-detail').hide();
    $('#build-steps').hide();
    $('#output').hide();
    var buildPanel = $('#build-panel').html('');
    buildPanel.addClass("panel panel-default");
    buildPanel.append('<div class="panel-heading"> Current Builds for <a href="' + repoData.url + '">' + repoData.name + '</a></div>');
    buildPanel.append('<div class="panel-body"><div id="build-area" class="table-responsive"></div></div>');
    $('#build-area').append('<table class="table table-striped"> <thead> <tr> <th>Result</th> <th>Build Id</th> <th>Time</th> <th>Committers</th> </tr> </thead><tbody id="all-builds"></tbody></table>');
    var table = $('#all-builds');
    repoData.builds.forEach(function (build) {
        var time = moment(build.timestamp).format('MMMM Do, HH:mm:ss');
        var buildRow = $('<tr></tr>');
        buildRow.append("<td><span class='glyphicon "
            + (getBuildStatusIcon(build.status, build.success))
            + "'></span></td>");
        buildRow.append('<td>' + build.number + '</a></td>');
        buildRow.append('<td>' + time + '</td>');
        buildRow.append('<td>' + build.committers + '</td>');
        buildRow.click(getBuildData.bind(null, repoData.id, build.number));
        table.append(buildRow);
    });
}

function getBuildStatusIcon(status, success) {
    if (status === "Done") {
		if (success) {
			return "glyphicon-ok"
		} else {
			return "glyphicon-remove"
		}
	} else {
		return "glyphicon-hourglass"
	}
}

function handleBuildClick(buildData) {
    $('#build-steps').show();
    $('#output').hide();
    var buildSteps = $('#build-steps').html('')
    buildSteps.addClass("panel panel-default");
    buildSteps.append('<div class="panel-heading"> <h3>Build Steps for ' + buildData.number + '</h3></div>');
    buildSteps.append('<div class="panel-body"><div id="build-step-area" class="table-responsive"></div></div>');
    $('#build-step-area').append('<table class="table table-striped"> <thead> <tr> <th>Status</th> <th>Step Name</th></tr> </thead><tbody id="all-steps"></tbody></table>');
    var table = $('#all-steps');
    buildData.steps.forEach(function (step) {
        var stepRow = $('<tr></tr>');

        stepRow.append("<td><span class='glyphicon "
            + (getBuildStatusIcon(step.status, step.exitcode == 0))
            + "'></span></td>");


        stepRow.append('<td>' + step.name + '</td>');
        stepRow.click(handleStepClick.bind(null, step));
        table.append(stepRow);
    });
}

var ws = null

function handleStepClick(step) {
    var ansi_up = new AnsiUp;
    var html = ansi_up.ansi_to_html(step.build).replace(/(?:\r\n|\r|\n)/g, '<br/>');
    var output = $('#output').html('');

    output.addClass("panel panel-default");
    output.append('<div class="panel-heading"> Build Output ' + step.name + '</div>');
    output.append('<div class="panel-body output-panelbody"><div id="output-log">' + html + '</div></div>');
    if (step.status === 'Running') {
        if (ws == null) {
            ws = new WebSocket("ws://localhost:8080/ws");
        }
        ws.onopen = function () {
            console.log('Connected')
        }

        ws.onmessage = function (evt) {
            var ansi_up = new AnsiUp;
            var html = ansi_up.ansi_to_html(JSON.parse(evt.data).Line);
            var out = $('#output-log');
            out.append('<div>' + html + '</div>');
            var outputlog = $('.output-panelbody')
            outputlog.scrollTop(outputlog[0].scrollHeight);
        }
    }
    output.show()
}

$(document).ready(getRepos);
