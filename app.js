window.onload = function() {
	// Get Username and User Type from current URL
	var userName = getParameterByName('username');
	var userType = getParameterByName('usertype');
	
	function getParameterByName(name, url) {
		if (!url) url = window.location.href;
		name = name.replace(/[\[\]]/g, "\\$&");
		var regex = new RegExp("[?&]" + name + "(=([^&#]*)|&|#|$)"),
			results = regex.exec(url);
		if (!results) return null;
		if (!results[2]) return '';
		return decodeURIComponent(results[2].replace(/\+/g, " "));
	}
	
	// Create a new WebSocket.
	var socket = new WebSocket('ws://192.168.1.101:8080/user');
	var socketStatus = document.getElementById('status');
	var currentPage = location.pathname.substring(location.pathname.lastIndexOf("/") + 1);
		
	// Handle any errors that occur.
	socket.onerror = function(error) {
		console.log('WebSocket Error: ' + error);
	};
	
	// Show a disconnected message when the WebSocket is closed.
	socket.onclose = function(event) {
		socketStatus.innerHTML = 'Disconnected from WebSocket.';
		socketStatus.className = 'closed';
	};
	
	// Close Web-Socket on Logout
	document.getElementById('logOutHref').onclick = function() {
		socket.close();
	};

	// Show a connected message when the WebSocket is opened.
	socket.onopen = function(event) {
		socketStatus.innerHTML = 'Connected to: ' + event.currentTarget.URL +
									'</br>User-Name: ' + userName +
									'</br>Page-Name: ' + currentPage + '</br>';
		
		if (currentPage == "overview.html") {
			var messageJSON = {
				"CustomerType": userType,
				"CustomerUname": userName,
				"Action": "getListOfContainers"
			} 

			var json = JSON.stringify(messageJSON);
		
			// Send the message through the WebSocket.
			socket.send(json);
		}
	};
	
	// Handle messages sent by the server.
	socket.onmessage = function(event) {
		var message = event.data;
		
		var serverResponse = JSON.parse(message)
		
		if (currentPage == "create_container.html") {
		
			var containerStatus =  document.getElementById('containerStatus');
			var containerIPaddress =  document.getElementById('ContainerIPaddress');
			
			if ( serverResponse.ContainerStatus == "Container Already Exists" ) {
				containerStatus.innerHTML = "Error: Container Already Exists";	
			}
	
			if ( serverResponse.ContainerIPaddress != null )	{
				socketStatus.innerHTML += serverResponse.ContainerName + ' Started Successfully </br>' +
											'Click this link to go to your website: <a href="http://' + 
											serverResponse.ContainerIPaddress + 
											'" target="_blank">' + serverResponse.ContainerName + '</a></br>';
			}
			
			if ( serverResponse.ContainerStatus == "Error Dialing Database Server" ) {
				socketStatus.innerHTML += '</br>Error: ' + serverResponse.ContainerStatus;
			}
			
			if ( serverResponse.Action == "Database Error" ) {
				socketStatus.innerHTML += '</br>Error: ' + serverResponse.ContainerStatus;
			}
			
			if ( serverResponse.Action == "LXC Server Dial Error" ) {
				socketStatus.innerHTML += '</br>Error: ' + serverResponse.ContainerStatus;
			}
		}
		
		var userContainerName = document.getElementById('userContainerName');
		var userContainerStatus = document.getElementById('userContainerStatus');
		var containerList = document.getElementById('containerForm-3');
		
		if (currentPage == "overview.html") {
		
			var messagesList = document.getElementById('messages');
			messagesList.innerHTML += '<li class="received"><span>Received:</span>' +
									   message + '</li>';
			
			if ( serverResponse.Action == "getListOfContainers" ) {
				if ( serverResponse.ContainerStatus != "Error Dialing Database Server" ) {
					
					userContainerName.innerHTML = serverResponse.ContainerName;

					userContainerStatus.innerHTML = serverResponse.ContainerStatus;
					
					containerList.innerHTML += '<table><tr>' + '<td align="left" width="425px">' + 
													serverResponse.ContainerName + '</td> ' +
													'<td align="center" width="100px">' + serverResponse.ContainerStatus + '</td>' +
													'<td align="center" width="100px"></td>' +
													'<td align="center" width="100px"></td>' +
													'<td align="center" width="100px"></td>' +
													'<td align="center" width="100px"></td>' +
													'<td align="center" width="100px"></td>' +
													'<td align="center" width="100px"></td>' +
													'<td align="center" width="100px"></td></tr></table>';	
				} else {
					socketStatus.innerHTML += '</br>Error' + serverResponse.ContainerName + ': ' + 
												serverResponse.ContainerStatus;
				}
			}
						
			if ( serverResponse.Action == "Database Error" ) {
				socketStatus.innerHTML += '</br>Error: ' + serverResponse.ContainerStatus;
			}
			
			if ( serverResponse.Action == "LXC Server Dial Error" ) {
				socketStatus.innerHTML += '</br>Error: ' + serverResponse.ContainerStatus;
			}
			
			if ( serverResponse.Action == "createDBentryForClone" ) {
				socketStatus.innerHTML += 'Clone: ' + serverResponse.ContainerName + ' - ' + serverResponse.ContainerStatus;
			}
			
			if ( serverResponse.Action == "createDBentryForSnaphot" ) {
				socketStatus.innerHTML += 'Snapshot: ' + serverResponse.ContainerName + ' - ' + serverResponse.ContainerStatus;
			}
			
			if ( serverResponse.Action == "UpdateContainerStatus" ) {
				if ( serverResponse.ContainerIPaddress != null ) {
					socketStatus.innerHTML += "</br>Update: " + serverResponse.ContainerName + 
												"</br>Status:  " + serverResponse.ContainerStatus +
												' ---- Click this link to go to your website: <a href="http://' 
												+ serverResponse.ContainerIPaddress + 
												'" target="_blank">' + serverResponse.ContainerName + '</a></br>';
											
				
					userContainerStatus.innerHTML = serverResponse.ContainerStatus;
				}
				
				if ( serverResponse.ContainerIPaddress == null ) {
					socketStatus.innerHTML += "</br>Update: " + serverResponse.ContainerName + 
													"</br>Status:  " + serverResponse.ContainerStatus + '</br>';
													
					userContainerStatus.innerHTML = serverResponse.ContainerStatus;
				}
			}
						
			if ( serverResponse.Action == "deleteContainerFromDatabase" ) {
				socketStatus.innerHTML += "</br>Update: " + serverResponse.ContainerName + 
											"</br>Status:  " + serverResponse.ContainerStatus + '</br>';
			
				userContainerStatus.innerHTML = serverResponse.ContainerStatus;
			}
		}
	};

				
	// Submit forms and send to Proxy-ControlServer, controls are based on current html page title
	if (currentPage == "overview.html") {
		// CReate href to include current Username and Usertype
		document.getElementById('createContainerHref').onclick = function() {
			socket.close();
			document.getElementById('createContainerHref').href = 'create_container.html?username=' + userName + '&usertype=' + userType;
		};
		
		var overviewFormA = document.getElementById('overview-form-1');
		var userContainerStatus = document.getElementById('userContainerStatus');

		overviewFormA.onsubmit = function(e) {
			e.preventDefault();
			
			var ovFormRdOptsA = document.getElementsByName('cStartStop-1');
			var ovFormConNameA = document.getElementById('containerForm').rows[2].cells.namedItem("allContainerName").innerHTML;
			var reqAction;
			 

			for (var i = 0, length = ovFormRdOptsA.length; i < length; i++) {
				if (ovFormRdOptsA[i].checked) {
					// do whatever you want with the checked radio
					alert(ovFormRdOptsA[i].value);
					break;
				}
			}
			var messageJSON = {
				"CustomerType": userType,
				"CustomerUname":userName,
				"Action": reqAction,
				"ContainerName": ovFormConNameA
			} 
					
			var json = JSON.stringify(messageJSON);

			// Testing Purposes Only
			var messagesList = document.getElementById('messages');

			messagesList.innerHTML += '<li><span>Sent:</span>' + json + '</li>';
									  
			// Send the message through the WebSocket.
			socket.send(json);
			
			return false;
		};
		
		var overviewFormB = document.getElementById('overview-form-2');
				
		overviewFormB.onsubmit = function(e) {
			e.preventDefault();
			
			var ovFormRdOptsB = document.getElementsByName('cStartStop-2');
			var ovFormConNameB = document.getElementById('containerForm-2').rows[2].cells.namedItem("userContainerName").innerHTML;
			var currentUserContainerStatus = document.getElementById('userContainerStatus').innerHTML;
			var reqAction;
			
			for (var i = 0, length = ovFormRdOptsB.length; i < length; i++) {
				if (ovFormRdOptsB[i].checked) {
					// do whatever you want with the checked radio
					switch ( ovFormRdOptsB[i].value ) {
						case "containerStart":
							if ( currentUserContainerStatus=="Running" ) {
								alert("The container is already running!!!");
								return false;
							} else {
								alert(ovFormRdOptsB[i].value);
							}
							break;
						case "containerStop":
							if ( currentUserContainerStatus=="Stopped" ) {
								alert("The container is already stopped!!!");
								return false;
							} else {
								alert(ovFormRdOptsB[i].value);
							}
							break;
						case "containerRestart":
							if ( currentUserContainerStatus=="Stopped" ) {
								alert("The container is already stopped!!!");
								return false;
							} else {
								alert(ovFormRdOptsB[i].value);
							}
							break;
						case "containerDelete":
							if ( currentUserContainerStatus=="Running" ) {
								alert("The container is Running, you will need to stop the container before deleting it");
								return false;
							} else {
								alert(ovFormRdOptsB[i].value);
							}
							break;
						case "containerBackUp":
							if ( currentUserContainerStatus=="Running" ) {
								alert("The container is Running, you will need to stop the container before backing it up");
								return false;
							} else {
								alert(ovFormRdOptsB[i].value);
							}
							break;
						case "containerSnapShot":
							if ( currentUserContainerStatus=="Running" ) {
								alert("The container is Running, you will need to stop the container before snapshotting it");
								return false;
							} else {
								alert(ovFormRdOptsB[i].value);
							}
							break;
						default:
							alert(ovFormRdOptsB[i].value);
					}
					alert(ovFormRdOptsB[i].value);
					break;
				}
			}
			
			var messageJSON = {
				"CustomerType": userType,
				"CustomerUname":userName,
				"Action": ovFormRdOptsB[i].value,
				"ContainerName": ovFormConNameB
			} 

			var json = JSON.stringify(messageJSON);

			// Testing Purposes Only
			var messagesList = document.getElementById('messages');

			messagesList.innerHTML += '<li><span>Sent:</span>' + json + '</li>';
									  
			// Send the message through the WebSocket.
			socket.send(json);
			return true;
		};
	}

	if (currentPage == "create_container.html") {
		document.getElementById('overViewHref').onclick = function() {
			socket.close();
			document.getElementById('overViewHref').href = 'overview.html?username=' + userName + '&usertype=' + userType;
		};
		
		// Deal with form values to create a new container
		// Send a message when the form is submitted.
		var newContainerForm = document.getElementById('newContainerForm');
		var cName = document.getElementById('containerName');
		var sName = document.getElementById('BaseServer');
		var cmsName = document.getElementById('CMS');
		var wsName = document.getElementById('websiteName');
		var DBrootPWD = document.getElementById('dbRootPWD');
		var DBadminUname = document.getElementById('dbAdminUname');
		var DBadminPWD = document.getElementById('dbAdminPwd');
	
		newContainerForm.onsubmit = function(e) {
			//e.preventDefault();
			
			var messageJSON = {
				"CustomerType": userType,
				"CustomerUname":userName,
				"Action": "createNew",
				"ContainerName": cName.value,
				"BaseServer": sName.value,
				"CMS": cmsName.value,
				"WebsiteName": wsName.value,
				"DBrootPWD": DBrootPWD.value,
				"DBadminUname": DBadminUname.value,
				"DBadminPWD": DBadminPWD.value
			} 

			var json = JSON.stringify(messageJSON);

			// Testing Purposes Only
			//var messagesList = document.getElementById('messages');
	
			//messagesList.innerHTML += '<li><span>Sent:</span>' + json + '</li>';
									  
			// Send the message through the WebSocket.
			socket.send(json);
		
			return false;
		};
	}
};
